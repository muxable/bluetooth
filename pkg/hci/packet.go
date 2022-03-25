package hci

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

type Packet interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

type CommandPacket interface {
	Packet
	Opcode() Opcode
}

func Unmarshal(buf []byte) (Packet, error) {
	if len(buf) == 0 {
		return nil, io.ErrShortBuffer
	}
	switch PacketType(buf[0]) {
	case PacketTypeCommand:
		p := &GenericCommandPacket{}
		if err := p.Unmarshal(buf[1:]); err != nil {
			return nil, err
		}
		return p, nil
	case PacketTypeEvent:
		s := uint8(buf[2])
		if len(buf) != int(s+3) {
			return nil, io.ErrShortBuffer
		}
		switch EventCode(buf[1]) {
		case EventCodeCommandComplete:
			p := &CommandCompleteEventPacket{}
			return p, p.Unmarshal(buf)
		case EventCodeLEMeta:
			switch LEMetaSubeventCode(buf[3]) {
			case LEMetaSubeventCodeConnectionComplete:
				p := &LEConnectionCompleteEventPacket{}
				if err := p.Unmarshal(buf); err != nil {
					return nil, err
				}
				return p, nil
			}
		case EventCodeNumberOfCompletedPackets:
			p := &NumberOfCompletedPacketsEventPacket{}
			return p, p.Unmarshal(buf)
		}
	case PacketTypeACLData:
		p := &ACLDataPacket{}
		if err := p.Unmarshal(buf); err != nil {
			return nil, err
		}
		return p, nil
	}
	return nil, errors.New("unsupported packet type")
}

type ACLDataPacket struct {
	PacketBoundaryFlag uint8
	BroadcastFlag      uint8
	ConnectionHandle   uint16
	Payload            []byte
}

func (p *ACLDataPacket) Unmarshal(buf []byte) error {
	if buf[0] != byte(PacketTypeACLData) {
		return errors.New("incorrect packet")
	}
	b := binary.LittleEndian.Uint16(buf[1:])
	p.PacketBoundaryFlag = byte((b >> 12) & 0x03)
	p.BroadcastFlag = byte((b >> 14) & 0x03)
	p.ConnectionHandle = b & 0x0FFF
	s := binary.LittleEndian.Uint16(buf[3:])
	if len(buf) != int(s+5) {
		return io.ErrShortBuffer
	}
	p.Payload = buf[5:]
	return nil
}

func (p *ACLDataPacket) Marshal() ([]byte, error) {
	buf := make([]byte, 5)
	buf[0] = byte(PacketTypeACLData)
	binary.LittleEndian.PutUint16(buf[1:], uint16(p.ConnectionHandle)|(uint16(p.PacketBoundaryFlag)<<12)|(uint16(p.BroadcastFlag)<<14))
	binary.LittleEndian.PutUint16(buf[3:], uint16(len(p.Payload)))
	return append(buf, p.Payload...), nil
}

// GenericCommandPacket encompasses many argument-less packets.
type GenericCommandPacket struct {
	opcode Opcode
}

func NewGenericCommandPacket(opcode Opcode) *GenericCommandPacket {
	return &GenericCommandPacket{opcode}
}

func (p *GenericCommandPacket) Marshal() ([]byte, error) {
	buf := make([]byte, 4)
	buf[0] = uint8(PacketTypeCommand)
	binary.LittleEndian.PutUint16(buf[1:], uint16(p.opcode))
	return buf, nil
}

func (p *GenericCommandPacket) Unmarshal(buf []byte) error {
	if buf[0] != byte(PacketTypeCommand) {
		return errors.New("incorrect packet")
	}
	if int(buf[3]) != 0 || len(buf) != 4 {
		return io.ErrShortBuffer
	}
	p.opcode = Opcode(binary.LittleEndian.Uint16(buf[1:2]))
	return nil
}

func (p *GenericCommandPacket) Opcode() Opcode {
	return p.opcode
}

type LESetAdvertisingEnableCommandPacket struct {
	AdvertisingEnable bool
}

func (p *LESetAdvertisingEnableCommandPacket) Marshal() ([]byte, error) {
	buf := make([]byte, 5)
	buf[0] = byte(PacketTypeCommand)
	binary.LittleEndian.PutUint16(buf[1:], uint16(OpcodeLESetAdvertisingEnable))
	buf[3] = 1
	if p.AdvertisingEnable {
		buf[4] = 1
	}
	return buf, nil
}

func (p *LESetAdvertisingEnableCommandPacket) Unmarshal(buf []byte) error {
	if buf[0] != byte(PacketTypeCommand) || binary.LittleEndian.Uint16(buf[1:]) != uint16(OpcodeLESetAdvertisingEnable) {
		return errors.New("incorrect packet")
	}
	if buf[3] != 1 || len(buf) != 5 {
		return io.ErrShortBuffer
	}
	p.AdvertisingEnable = buf[4] == 1
	return nil
}

func (p *LESetAdvertisingEnableCommandPacket) Opcode() Opcode {
	return OpcodeLESetAdvertisingEnable
}

type CommandCompleteEventPacket struct {
	NumCommandPackets uint8
	CommandOpcode     Opcode
	ReturnParameters  []byte
}

func (p *CommandCompleteEventPacket) Unmarshal(buf []byte) error {
	if buf[0] != byte(PacketTypeEvent) || buf[1] != byte(EventCodeCommandComplete) {
		return errors.New("incorrect packet")
	}
	s := int(buf[2])
	if len(buf) != s+3 {
		return io.ErrShortBuffer
	}
	p.NumCommandPackets = buf[3]
	p.CommandOpcode = Opcode(binary.LittleEndian.Uint16(buf[4:]))
	p.ReturnParameters = buf[6:]
	return nil
}

func (p *CommandCompleteEventPacket) Marshal() ([]byte, error) {
	if len(p.ReturnParameters)+2 > math.MaxUint8 {
		return nil, io.ErrShortWrite
	}
	buf := make([]byte, 6+len(p.ReturnParameters))
	buf[0] = byte(PacketTypeEvent)
	buf[1] = byte(EventCodeCommandComplete)
	buf[2] = byte(len(p.ReturnParameters) + 2)
	buf[3] = byte(p.NumCommandPackets)
	binary.LittleEndian.PutUint16(buf[4:], uint16(p.CommandOpcode))
	copy(buf[6:], p.ReturnParameters)
	return buf, nil
}

type NumberOfCompletedPacketsEventPacket struct {
	NumHandles          uint8
	ConnectionHandles   []uint16
	NumCompletedPackets []uint16
}

func (p *NumberOfCompletedPacketsEventPacket) Unmarshal(buf []byte) error {
	if buf[0] != byte(PacketTypeEvent) || buf[1] != byte(EventCodeNumberOfCompletedPackets) {
		return errors.New("incorrect packet")
	}
	s := int(buf[2])
	if len(buf) != s+3 {
		return io.ErrShortBuffer
	}
	p.NumHandles = buf[3]
	p.ConnectionHandles = make([]uint16, p.NumHandles)
	p.NumCompletedPackets = make([]uint16, p.NumHandles)
	for i := 0; i < int(p.NumHandles); i++ {
		p.ConnectionHandles[i] = binary.LittleEndian.Uint16(buf[4+i*2 : 4+i*2+2])
		p.NumCompletedPackets[i] = binary.LittleEndian.Uint16(buf[4+int(p.NumHandles)+i*2 : 4+int(p.NumHandles)+i*2+2])
	}
	return nil
}

func (p *NumberOfCompletedPacketsEventPacket) Marshal() ([]byte, error) {
	if len(p.ConnectionHandles) != int(p.NumHandles) || len(p.NumCompletedPackets) != int(p.NumHandles) {
		return nil, io.ErrShortWrite
	}
	buf := make([]byte, 4+len(p.ConnectionHandles)+len(p.NumCompletedPackets)*2)
	buf[0] = byte(PacketTypeEvent)
	buf[1] = byte(EventCodeNumberOfCompletedPackets)
	buf[2] = byte(len(p.ConnectionHandles) + len(p.NumCompletedPackets))
	buf[3] = byte(p.NumHandles)
	for i := 0; i < int(p.NumHandles); i++ {
		binary.LittleEndian.PutUint16(buf[4+i*2:], p.ConnectionHandles[i])
		binary.LittleEndian.PutUint16(buf[4+int(p.NumHandles)+i*2:], p.NumCompletedPackets[i])
	}
	return buf, nil
}

type Role uint8

const (
	RoleCentral    Role = 0
	RolePeripheral Role = 1
)

type CentralClockAccuracy uint8

const (
	CentralClockAccuracy500PPM CentralClockAccuracy = 0
	CentralClockAccuracy250PPM CentralClockAccuracy = 1
	CentralClockAccuracy150PPM CentralClockAccuracy = 2
	CentralClockAccuracy100PPM CentralClockAccuracy = 3
	CentralClockAccuracy75PPM  CentralClockAccuracy = 4
	CentralClockAccuracy50PPM  CentralClockAccuracy = 5
	CentralClockAccuracy30PPM  CentralClockAccuracy = 6
	CentralClockAccuracy20PPM  CentralClockAccuracy = 7
)

type LEConnectionCompleteEventPacket struct {
	ConnectionHandle     uint16
	Role                 Role
	PeerAddressType      PeerAddressType
	PeerAddress          BDAddr
	ConnectionInterval   uint16
	PeripheralLatency    uint16
	SupervisionTimeout   uint16
	CentralClockAccuracy CentralClockAccuracy
}

func (p *LEConnectionCompleteEventPacket) Marshal() ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (p *LEConnectionCompleteEventPacket) Unmarshal(buf []byte) error {
	if buf[0] != byte(PacketTypeEvent) || buf[1] != byte(EventCodeLEMeta) {
		return errors.New("incorrect packet")
	}
	if buf[2] != 19 || len(buf) != 22 {
		return io.ErrShortBuffer
	}
	if buf[3] != byte(LEMetaSubeventCodeConnectionComplete) {
		return errors.New("incorrect subevent")
	}
	if buf[4] != 0 {
		return errors.New("unexpected status")
	}
	p.ConnectionHandle = binary.LittleEndian.Uint16(buf[5:7])
	p.Role = Role(buf[7])
	p.PeerAddressType = PeerAddressType(buf[8])
	copy(p.PeerAddress[:], buf[9:15])
	p.ConnectionInterval = binary.LittleEndian.Uint16(buf[15:17])
	p.PeripheralLatency = binary.LittleEndian.Uint16(buf[17:19])
	p.SupervisionTimeout = binary.LittleEndian.Uint16(buf[19:21])
	p.CentralClockAccuracy = CentralClockAccuracy(buf[21])
	return nil
}
