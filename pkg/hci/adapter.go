package hci

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/google/uuid"
)

type Adapter struct {
	*Socket

	onPacket map[string]func(Packet, error)
}

func NewConn(s *Socket) *Adapter {
	a := &Adapter{Socket: s, onPacket: make(map[string]func(Packet, error))}
	go func() {
		for {
			p, err := a.ReadPacket()
			if err != nil {
				for _, cb := range a.onPacket {
					cb(nil, err)
				}
				return
			}
			for _, cb := range a.onPacket {
				cb(p, nil)
			}
		}
	}()
	return a
}

func (a *Adapter) op(p CommandPacket) ([]byte, error) {
	done := make(chan []byte)
	defer close(done)
	id := uuid.NewString()
	a.onPacket[id] = func(q Packet, err error) {
		if err != nil {
			done <- nil
			return
		}
		switch q := q.(type) {
		case *CommandCompleteEventPacket:
			if q.CommandOpcode != p.Opcode() {
				return
			}
			delete(a.onPacket, id)
			done <- q.ReturnParameters
		}
	}
	if err := a.WritePacket(p); err != nil {
		return nil, err
	}
	return <-done, nil
}

func (a *Adapter) Reset() error {
	buf, err := a.op(NewGenericCommandPacket(OpcodeReset))
	if err != nil {
		return err
	}
	if buf[0] != 0 {
		return errors.New("command failed")
	}
	return err
}

func (a *Adapter) ReadBDAddr() (BDAddr, error) {
	var addr BDAddr
	buf, err := a.op(NewGenericCommandPacket(OpcodeReadBDAddr))
	if err != nil {
		return addr, err
	}
	if buf[0] != 0 {
		return addr, errors.New("command failed")
	}
	if copy(addr[:], buf[1:]) != 6 {
		return addr, io.ErrShortWrite
	}
	return addr, nil
}

func (a *Adapter) ClearFilterAcceptList() error {
	buf, err := a.op(NewGenericCommandPacket(OpcodeClearFilterAcceptList))
	if err != nil {
		return err
	}
	if buf[0] != 0 {
		return errors.New("command failed")
	}
	return err
}

func (a *Adapter) ReadFilterAcceptListSize() (uint8, error) {
	buf, err := a.op(NewGenericCommandPacket(OpcodeReadFilterAcceptListSize))
	if err != nil {
		return 0, err
	}
	if buf[0] != 0 {
		return 0, errors.New("command failed")
	}
	return buf[1], nil
}

type LEReadBufferSizeResponse struct {
	LEACLDataPacketLength    uint16
	TotalNumLEACLDataPackets uint8
	ISODataPacketLength      uint16
	TotalNumISODataPackets   uint8
}

func (a *Adapter) LEReadBufferSize() (*LEReadBufferSizeResponse, error) {
	buf, err := a.op(NewGenericCommandPacket(OpcodeLEReadBufferSize))
	if err != nil {
		return nil, err
	}
	if buf[0] != 0 {
		return nil, errors.New("command failed")
	}
	r := &LEReadBufferSizeResponse{
		LEACLDataPacketLength:    binary.LittleEndian.Uint16(buf[1:3]),
		TotalNumLEACLDataPackets: buf[3],
	}
	if len(buf) > 4 {
		r.ISODataPacketLength = binary.LittleEndian.Uint16(buf[4:6])
		r.TotalNumISODataPackets = buf[6]
	}
	return r, nil
}

type LESupportedStates uint64

func (a *Adapter) LEReadSupportedStates() (LESupportedStates, error) {
	buf, err := a.op(NewGenericCommandPacket(OpcodeLEReadSupportedStates))
	if err != nil {
		return 0, err
	}
	if buf[0] != 0 {
		return 0, errors.New("command failed")
	}
	return LESupportedStates(binary.LittleEndian.Uint64(buf[1:9])), nil
}

func (a *Adapter) LESetAdvertisingEnable(enable bool) error {
	buf, err := a.op(&LESetAdvertisingEnableCommandPacket{AdvertisingEnable: enable})
	if err != nil {
		return err
	}
	if buf[0] != 0 {
		return errors.New("command failed")
	}
	return nil
}

type Conn struct {
	*Adapter

	ConnectionHandle     uint16
	Role                 Role
	PeerAddressType      PeerAddressType
	PeerAddress          BDAddr
	ConnectionInterval   uint16
	PeripheralLatency    uint16
	SupervisionTimeout   uint16
	CentralClockAccuracy CentralClockAccuracy

	bufCh chan []byte
	errCh chan error
}

func (a *Adapter) Accept() (*Conn, error) {
	conn := make(chan *Conn)
	defer close(conn)
	errch := make(chan error)
	defer close(errch)
	id := uuid.NewString()
	a.onPacket[id] = func(p Packet, err error) {
		if err != nil {
			errch <- err
			return
		}
		switch p := p.(type) {
		case *LEConnectionCompleteEventPacket:
			delete(a.onPacket, id)
			c := &Conn{
				Adapter:              a,
				ConnectionHandle:     p.ConnectionHandle,
				Role:                 p.Role,
				PeerAddressType:      p.PeerAddressType,
				PeerAddress:          p.PeerAddress,
				ConnectionInterval:   p.ConnectionInterval,
				PeripheralLatency:    p.PeripheralLatency,
				SupervisionTimeout:   p.SupervisionTimeout,
				CentralClockAccuracy: p.CentralClockAccuracy,
				bufCh:                make(chan []byte),
				errCh:                make(chan error),
			}
			go func() {
				cid := uuid.NewString()
				var buf []byte
				a.onPacket[cid] = func(q Packet, err error) {
					if err != nil {
						c.errCh <- err
						return
					}
					switch q := q.(type) {
					case *ACLDataPacket:
						if q.ConnectionHandle != p.ConnectionHandle {
							// this packet is for another connection.
							return
						}
						switch q.PacketBoundaryFlag {
						case 0b01: // continuation packet
							buf = append(buf, q.Payload...)
						case 0b10: // start packet
							if len(buf) > 0 {
								c.errCh <- errors.New("unexpected start packet")
								return
							}
							buf = q.Payload
						default:
							// unhandled packet type
							c.errCh <- errors.New("unhandled packet type")
							return
						}
						// introspect the packet to see if we're done
						if len(buf) >= 4 && len(buf) == int(binary.LittleEndian.Uint16(buf[:2]))+4 {
							// this packet is complete
							c.bufCh <- buf
							buf = nil
						}
					}
				}
			}()
			conn <- c
		}
	}
	select {
	case c := <-conn:
		return c, nil
	case err := <-errch:
		return nil, err
	}
}

func (c *Conn) Read(buf []byte) (int, error) {
	select {
	case b := <-c.bufCh:
		n := copy(buf, b)
		if n < len(b) {
			return n, io.ErrShortBuffer
		}
		return len(b), nil
	case err := <-c.errCh:
		return 0, err
	}
}