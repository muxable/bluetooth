package l2cap

import (
	"encoding/binary"
	"io"
	"sync"

	"github.com/muxable/bluetooth/pkg/hci"
)

type ConnectionOrientedChannel struct {
	L2CAPConn *Conn

	Identifier uint8 // the identifier of the original request.

	PSM            uint16
	RxCID          ChannelID
	TxCID          ChannelID
	TxMPS          uint16
	RxMPS          uint16
	TxMTU          uint16
	RxMTU          uint16
	TxCredits      uint16
	RxCredits      uint16
	RxBuf          []byte
	RxSDUBytesLeft uint16

	writeMutex        sync.Mutex // this is necessary to prevent writes from interfering with each other.
	txCreditSemaphore *sync.Cond
	rxCh              chan []byte
}

// this api is a bit strange because we need to actually ack connections.

func (c *ConnectionOrientedChannel) Approve(initiallyPaused bool, rxMTU uint16) error {
	c.RxMTU = rxMTU
	c.RxMPS = rxMTU
	if c.RxMPS > 1004 {
		c.RxMPS = 1004
	}
	c.RxCredits = 500

	r := &LECreditBasedConnectionResponsePacket{
		Identifier:     c.Identifier,
		DestinationCID: c.RxCID,
		MTU:            c.RxMTU,
		MPS:            c.RxMPS,
		InitialCredits: c.RxCredits,
		Result:         LECreditBasedConnectionResultSuccessful,
	}

	c.L2CAPConn.cocs[c.RxCID] = c

	return c.L2CAPConn.writeSignallingPacket(ChannelIDSignallingLEU, r)
}

func (c *ConnectionOrientedChannel) Reject(result LECreditBasedConnectionResult) error {
	r := &LECreditBasedConnectionResponsePacket{
		Identifier: c.Identifier,
		Result:     result,
	}

	return c.L2CAPConn.writeSignallingPacket(ChannelIDSignallingLEU, r)
}

func (c *ConnectionOrientedChannel) receive(buf []byte) error {
	if c.RxCredits == 0 || (c.RxSDUBytesLeft == 0 && len(buf) < 2) {
		return c.L2CAPConn.writeSignallingPacket(ChannelIDSignallingLEU, &DisconnectionRequestPacket{
			Identifier:     NextIdentifier(),
			DestinationCID: c.TxCID,
			SourceCID:      c.RxCID,
		})
	}

	c.RxCredits--

	if c.RxSDUBytesLeft == 0 {
		c.RxSDUBytesLeft = binary.LittleEndian.Uint16(buf[0:2])
		buf = buf[2:] // remove the 2 bytes sdu length.
	}

	if len(buf) > int(c.RxSDUBytesLeft) || len(buf) > int(c.RxMPS) || c.RxSDUBytesLeft > c.RxMTU {
		return c.L2CAPConn.writeSignallingPacket(ChannelIDSignallingLEU, &DisconnectionRequestPacket{
			Identifier:     NextIdentifier(),
			DestinationCID: c.TxCID,
			SourceCID:      c.RxCID,
		})
	}

	c.RxBuf = append(c.RxBuf, buf...)
	c.RxSDUBytesLeft -= uint16(len(buf))

	if c.RxSDUBytesLeft == 0 {
		c.rxCh <- c.RxBuf
		c.RxBuf = nil
	}
	// assign new credits if necessary
	if c.RxCredits <= 70 {
		c.RxCredits += 500
		if err := c.L2CAPConn.writeSignallingPacket(ChannelIDSignallingLEU, &FlowControlCreditIndicationPacket{
			Identifier: NextIdentifier(),
			CID:        c.RxCID,
			Credits:    500,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *ConnectionOrientedChannel) Read(buf []byte) (int, error) {
	b, ok := <-c.rxCh
	if !ok {
		return 0, io.EOF
	}
	if len(buf) < len(b) {
		return 0, io.ErrShortBuffer
	}
	return copy(buf, b), nil
}

func (c *ConnectionOrientedChannel) Write(buf []byte) (int, error) {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	buf = append([]byte{0, 0}, buf...)
	binary.LittleEndian.PutUint16(buf, uint16(len(buf)-2))
	f := &BFrame{ChannelID: c.TxCID, Payload: buf}
	fbuf, err := f.Marshal()
	if err != nil {
		return 0, err
	}
	for i := 0; i < len(fbuf); i += int(c.TxMPS) {
		c.txCreditSemaphore.L.Lock()
		for c.TxCredits == 0 {
			c.txCreditSemaphore.Wait()
		}
		c.txCreditSemaphore.L.Unlock()

		c.TxCredits--

		var packetBoundary uint8
		if i > 0 {
			packetBoundary = 0x01
		}

		j := i + int(c.TxMPS)
		if j > len(fbuf) {
			j = len(fbuf)
		}

		if err := c.L2CAPConn.HCIConn.Socket.WritePacket(&hci.ACLDataPacket{
			ConnectionHandle:   c.L2CAPConn.HCIConn.ConnectionHandle,
			PacketBoundaryFlag: packetBoundary,
			Payload:            fbuf[i:j],
		}); err != nil {
			return 0, err
		}
	}
	return len(buf), nil
}

func (c *ConnectionOrientedChannel) Close() error {
	close(c.rxCh)
	return nil
}
