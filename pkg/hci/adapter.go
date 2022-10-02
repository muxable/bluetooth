package hci

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"

	"github.com/google/uuid"
)

type Adapter struct {
	*Socket

	onPacketLock sync.Mutex
	onPacket map[string]func(Packet, error)

	ACLMTU                  uint16
	ACLPacketsRemaining     uint16
	ACLPacketsRemainingCond *sync.Cond
	ACLPacketsPending       map[uint16]uint16
}

func NewConn(s *Socket) *Adapter {
	a := &Adapter{
		Socket:                  s,
		onPacket:                make(map[string]func(Packet, error)),
		ACLMTU:                  1023,
		ACLPacketsRemainingCond: sync.NewCond(&sync.Mutex{}),
		ACLPacketsPending:       make(map[uint16]uint16),
	}
	go func() {
		for {
			p, err := a.ReadPacket()
			if err != nil {
				a.onPacketLock.Lock()
				for _, cb := range a.onPacket {
					go cb(nil, err)
				}
				a.onPacketLock.Unlock()
				return
			}
			switch p := p.(type) {
			case *NumberOfCompletedPacketsEventPacket:
				a.ACLPacketsRemainingCond.L.Lock()
				for i := 0; i < int(p.NumHandles); i++ {
					a.ACLPacketsRemaining += p.NumCompletedPackets[i]
				}
				a.ACLPacketsRemainingCond.Broadcast()
				a.ACLPacketsRemainingCond.L.Unlock()
			case *DisconnectionCompleteEventPacket:
				a.ACLPacketsRemainingCond.L.Lock()
				a.ACLPacketsRemaining += a.ACLPacketsPending[p.ConnectionHandle]
				delete(a.ACLPacketsPending, p.ConnectionHandle)
				a.ACLPacketsRemainingCond.Broadcast()
				a.ACLPacketsRemainingCond.L.Unlock()
			}
			a.onPacketLock.Lock()
			for _, cb := range a.onPacket {
				go cb(p, nil)
			}
			a.onPacketLock.Unlock()
		}
	}()
	return a
}

func (a *Adapter) op(p CommandPacket) ([]byte, error) {
	done := make(chan []byte)
	defer close(done)
	id := uuid.NewString()
	a.onPacketLock.Lock()
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
			a.onPacketLock.Lock()
			delete(a.onPacket, id)
			a.onPacketLock.Unlock()
			done <- q.ReturnParameters
		}
	}
	a.onPacketLock.Unlock()
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
	a.ACLMTU = r.LEACLDataPacketLength

	a.ACLPacketsRemainingCond.L.Lock()
	a.ACLPacketsRemaining = uint16(r.TotalNumLEACLDataPackets)
	a.ACLPacketsRemainingCond.Broadcast()
	a.ACLPacketsRemainingCond.L.Unlock()
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
	a.onPacketLock.Lock()
	a.onPacket[id] = func(p Packet, err error) {
		if err != nil {
			errch <- err
			return
		}
		switch p := p.(type) {
		case *LEConnectionCompleteEventPacket:
			a.onPacketLock.Lock()
			delete(a.onPacket, id)
			a.onPacketLock.Unlock()
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
				a.onPacketLock.Lock()
				a.onPacket[cid] = func(q Packet, err error) {
					if err != nil {
						c.errCh <- err
						return
					}
					switch q := q.(type) {
					case *DisconnectionCompleteEventPacket:
						if q.ConnectionHandle == c.ConnectionHandle {
							a.onPacketLock.Lock()
							delete(a.onPacket, cid)
							a.onPacketLock.Unlock()
						}
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
				a.onPacketLock.Unlock()
			}()
			conn <- c
		}
	}
	a.onPacketLock.Unlock()
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

func (c *Conn) Write(buf []byte) (int, error) {
	for i := 0; i < len(buf); i += int(c.ACLMTU) {
		var pb uint8
		if i > 0 {
			pb = 1
		}

		j := i + int(c.ACLMTU)
		if j > len(buf) {
			j = len(buf)
		}

		p := &ACLDataPacket{
			ConnectionHandle:   c.ConnectionHandle,
			PacketBoundaryFlag: pb,
			Payload:            buf[i:j],
		}
		if err := c.WritePacket(p); err != nil {
			return 0, err
		}
	}
	return len(buf), nil
}

func (c *Conn) WritePacket(p Packet) error {
	c.ACLPacketsRemainingCond.L.Lock()
	for c.ACLPacketsRemaining == 0 {
		c.ACLPacketsRemainingCond.Wait()
	}
	c.ACLPacketsRemaining--
	c.ACLPacketsPending[c.ConnectionHandle]++
	c.ACLPacketsRemainingCond.L.Unlock()
	return c.Socket.WritePacket(p)
}
