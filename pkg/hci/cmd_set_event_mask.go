package hci

import (
	"encoding/binary"
	"errors"
	"io"
)

// Section 7.3.1
type EventMask uint64

const (
	EventMaskDisconnectionCompleteEvent        EventMask = (1 << 4)
	EventMaskEncryptionChangeEvent             EventMask = (1 << 7)
	EventMaskHardwareErrorEvent                EventMask = (1 << 15)
	EventMaskEncryptionKeyRefreshCompleteEvent EventMask = (1 << 47)
	EventMaskLEMetaEvent                       EventMask = (1 << 61)
)

type HCISetEventMaskCommandPacket struct {
	EventMask
}

func (p *HCISetEventMaskCommandPacket) Marshal() ([]byte, error) {
	buf := make([]byte, 12)
	buf[0] = byte(PacketTypeCommand)
	binary.LittleEndian.PutUint16(buf[1:], uint16(OpcodeSetEventMask))
	buf[3] = 8
	binary.LittleEndian.PutUint64(buf[4:], uint64(p.EventMask))
	return buf, nil
}

func (p *HCISetEventMaskCommandPacket) Unmarshal(buf []byte) error {
	if buf[0] != byte(PacketTypeCommand) || binary.LittleEndian.Uint16(buf[1:]) != uint16(OpcodeSetEventMask) {
		return errors.New("incorrect packet")
	}
	if buf[4] != 8 || len(buf) != 12 {
		return io.ErrShortBuffer
	}
	p.EventMask = EventMask(binary.LittleEndian.Uint64(buf[4:]))
	return nil
}

func (p *HCISetEventMaskCommandPacket) Opcode() Opcode {
	return OpcodeSetEventMask
}

func (a *Adapter) SetEventMask(mask EventMask) error {
	buf, err := a.op(&HCISetEventMaskCommandPacket{EventMask: mask})
	if err != nil {
		return err
	}
	if buf[0] != 0 {
		return errors.New("command failed")
	}
	return nil
}
