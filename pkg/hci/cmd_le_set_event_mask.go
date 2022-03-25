package hci

import (
	"encoding/binary"
	"errors"
	"io"
)

// Section 7.8.1
type LEEventMask uint64

const (
	LEEventMaskConnectionCompleteEvent             LEEventMask = (1 << 0)
	LEEventMaskAdvertisingReportEvent              LEEventMask = (1 << 1)
	LEEventMaskConnectionUpdateCompleteEvent       LEEventMask = (1 << 2)
	LEEventMaskReadRemoteUsedFeaturesCompleteEvent LEEventMask = (1 << 3)
	LEEventMaskLongTermKeyRequestEvent             LEEventMask = (1 << 4)
)

type HCILESetEventMaskCommandPacket struct {
	LEEventMask
}

func (p *HCILESetEventMaskCommandPacket) Marshal() ([]byte, error) {
	buf := make([]byte, 12)
	buf[0] = byte(PacketTypeCommand)
	binary.LittleEndian.PutUint16(buf[1:], uint16(OpcodeLESetEventMask))
	buf[3] = 8
	binary.LittleEndian.PutUint64(buf[4:], uint64(p.LEEventMask))
	return buf, nil
}

func (p *HCILESetEventMaskCommandPacket) Unmarshal(buf []byte) error {
	if buf[0] != byte(PacketTypeCommand) || binary.LittleEndian.Uint16(buf[1:]) != uint16(OpcodeLESetEventMask) {
		return errors.New("incorrect packet")
	}
	if buf[4] != 8 || len(buf) != 12 {
		return io.ErrShortBuffer
	}
	p.LEEventMask = LEEventMask(binary.LittleEndian.Uint64(buf[4:]))
	return nil
}

func (p *HCILESetEventMaskCommandPacket) Opcode() Opcode {
	return OpcodeLESetEventMask
}

func (a *Adapter) LESetEventMask(mask LEEventMask) error {
	buf, err := a.op(&HCILESetEventMaskCommandPacket{LEEventMask: mask})
	if err != nil {
		return err
	}
	if buf[0] != 0 {
		return errors.New("command failed")
	}
	return nil
}
