package l2cap

import (
	"encoding/binary"
	"io"
	"sync"
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

type ApprovedConnectionOrientedChannel struct {
	*ConnectionOrientedChannel
}

// this api is a bit strange because we need to actually ack connections.

func (c *ConnectionOrientedChannel) Approve(initiallyPaused bool, rxMTU uint16) (*ApprovedConnectionOrientedChannel, error) {
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

	a := &ApprovedConnectionOrientedChannel{ConnectionOrientedChannel: c}

	c.L2CAPConn.cocs[c.RxCID] = a

	return a, c.L2CAPConn.writeSignallingPacket(ChannelIDSignallingLEU, r)
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

func (c *ApprovedConnectionOrientedChannel) Read(buf []byte) (int, error) {
	b, ok := <-c.rxCh
	if !ok {
		return 0, io.EOF
	}
	if len(buf) < len(b) {
		return 0, io.ErrShortBuffer
	}
	return copy(buf, b), nil
}

func (c *ApprovedConnectionOrientedChannel) Write(buf []byte) (int, error) {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	sdu := append([]byte{0, 0}, buf...)
	binary.LittleEndian.PutUint16(sdu, uint16(len(sdu)-2))
	for i := 0; i < len(sdu); i += int(c.TxMPS) {
		c.txCreditSemaphore.L.Lock()
		for c.TxCredits == 0 {
			c.txCreditSemaphore.Wait()
		}
		c.txCreditSemaphore.L.Unlock()

		c.TxCredits--

		j := i + int(c.TxMPS)
		if j > len(sdu) {
			j = len(sdu)
		}
		
		f := &BFrame{ChannelID: c.TxCID, Payload: sdu[i:j]}
		fbuf, err := f.Marshal()
		if err != nil {
			return 0, err
		}

		if _, err := c.L2CAPConn.HCIConn.Write(fbuf); err != nil {
			return 0, err
		}
	}
	return len(buf), nil
}

func (c *ApprovedConnectionOrientedChannel) Close() error {
	close(c.rxCh)
	return nil
}
