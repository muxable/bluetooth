package l2cap

import (
	"errors"
	"math"
	"sync"

	"github.com/muxable/bluetooth/pkg/hci"
	"go.uber.org/zap"
)

type Conn struct {
	HCIConn *hci.Conn

	cocs          map[ChannelID]*ApprovedConnectionOrientedChannel
	nextChannelID ChannelID
}

func NewConn(conn *hci.Conn) *Conn {
	c := &Conn{
		HCIConn:       conn,
		cocs:          make(map[ChannelID]*ApprovedConnectionOrientedChannel),
		nextChannelID: 0x40,
	}
	return c
}

func (c *Conn) Accept() (*ConnectionOrientedChannel, error) {
	for {
		f, err := c.ReadFrame()
		if err != nil {
			return nil, err
		}

		switch f := f.(type) {
		case *BFrame:
			switch f.ChannelID {
			case ChannelIDAttributeProtocol:
				opcode := f.Payload[0]
				if opcode == 0x08 {
					// this is an attribute request, return with not found since we don't have a gatt db.
					if err := c.HCIConn.WritePacket(&hci.ACLDataPacket{
						ConnectionHandle: c.HCIConn.ConnectionHandle,
						Payload:          []byte{0x05, 0x00, 0x04, 0x00, 0x01, 0x08, 0x01, 0x00, 0x0a},
					}); err != nil {
						return nil, err
					}
				}
			case ChannelIDSignallingLEU:
				p, err := UnmarshalSignallingPacket(f.Payload)
				if err != nil {
					// this is an internal error that we should handle.
					return nil, err
				}
				switch p := p.(type) {
				case *LECreditBasedConnectionRequestPacket:
					if p.MTU < 23 || p.MPS < 23 || p.MPS > 65533 {
						r := &LECreditBasedConnectionResponsePacket{
							Identifier: p.Identifier,
							Result:     LECreditBasedConnectionResultRefusedUnacceptableParameters,
						}
						if err := c.writeSignallingPacket(ChannelIDSignallingLEU, r); err != nil {
							return nil, err
						}
					}
					mps := p.MPS
					if mps > 1004 {
						mps = 1004
					}
					if len(c.cocs) == 0xFFC0 {
						r := &LECreditBasedConnectionResponsePacket{
							Identifier: p.Identifier,
							Result:     LECreditBasedConnectionResultRefusedNoResourcesAvailable,
						}
						if err := c.writeSignallingPacket(ChannelIDSignallingLEU, r); err != nil {
							return nil, err
						}
					}
					if p.SourceCID <= 0x003F {
						r := &LECreditBasedConnectionResponsePacket{
							Identifier: p.Identifier,
							Result:     LECreditBasedConnectionResultRefusedInvalidSourceCID,
						}
						if err := c.writeSignallingPacket(ChannelIDSignallingLEU, r); err != nil {
							return nil, err
						}
					}
					for _, channel := range c.cocs {
						if channel.TxCID == p.SourceCID {
							r := &LECreditBasedConnectionResponsePacket{
								Identifier: p.Identifier,
								Result:     LECreditBasedConnectionResultRefusedSourceCIDAlreadyAllocated,
							}
							if err := c.writeSignallingPacket(ChannelIDSignallingLEU, r); err != nil {
								return nil, err
							}
						}
					}

					ch := &ConnectionOrientedChannel{
						L2CAPConn:         c,
						Identifier:        p.Identifier,
						PSM:               p.SPSM,
						RxCID:             c.nextChannelID,
						TxCID:             p.SourceCID,
						TxMPS:             p.MPS,
						TxMTU:             p.MTU,
						TxCredits:         p.InitialCredits,
						rxCh:              make(chan []byte),
						txCreditSemaphore: sync.NewCond(&sync.Mutex{}),
					}

					c.nextChannelID++

					return ch, nil

				case *FlowControlCreditIndicationPacket:
					if p.Credits == 0 {
						break
					}
					for _, ch := range c.cocs {
						if ch.TxCID != p.CID {
							continue
						}
						ch.txCreditSemaphore.L.Lock()
						if int(ch.TxCredits)+int(p.Credits) > math.MaxUint16 {
							if err := c.writeSignallingPacket(ChannelIDSignallingLEU, &DisconnectionRequestPacket{
								Identifier:     NextIdentifier(),
								DestinationCID: ch.TxCID,
								SourceCID:      ch.RxCID,
							}); err != nil {
								return nil, err
							}
						} else {
							ch.TxCredits += p.Credits
						}
						ch.txCreditSemaphore.Broadcast()
						ch.txCreditSemaphore.L.Unlock()

					}

				case *DisconnectionRequestPacket:
					r := &DisconnectionResponsePacket{
						Identifier:     p.Identifier,
						DestinationCID: p.DestinationCID,
						SourceCID:      p.SourceCID,
					}
					if err := c.writeSignallingPacket(ChannelIDSignallingLEU, r); err != nil {
						return nil, err
					}
					if err := c.cocs[ChannelID(p.DestinationCID)].Close(); err != nil {
						return nil, err
					}
				case *DisconnectionResponsePacket:
					if err := c.cocs[ChannelID(p.DestinationCID)].Close(); err != nil {
						return nil, err
					}
				default:
					// this is an internal error that we should handle.
					return nil, errors.New("unhandled packet type")
				}
			default:
				if coc, ok := c.cocs[f.ChannelID]; ok {
					coc.receive(f.Payload)
				} else {
					zap.L().Warn("received packet for unknown channel", zap.Uint16("channel", uint16(f.ChannelID)))
				}
				// this is an external channel.
			}
		}
	}
}

func (c *Conn) ReadFrame() (Frame, error) {
	buf := make([]byte, math.MaxUint16)
	n, err := c.HCIConn.Read(buf)
	if err != nil {
		return nil, err
	}
	return UnmarshalFrame(buf[:n])
}

func (c *Conn) writeSignallingPacket(channelID ChannelID, p SignallingPacket) error {
	pbuf, err := p.Marshal()
	if err != nil {
		return err
	}
	// assume this won't fragment.
	f := &BFrame{ChannelID: channelID, Payload: pbuf}
	fbuf, err := f.Marshal()
	if err != nil {
		return err
	}
	return c.HCIConn.WritePacket(&hci.ACLDataPacket{
		ConnectionHandle: c.HCIConn.ConnectionHandle,
		Payload:          fbuf,
	})
}
