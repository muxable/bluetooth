package l2cap

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

type Frame interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

func UnmarshalFrame(buf []byte) (Frame, error) {
	if len(buf) < 4 {
		return nil, io.ErrShortBuffer
	}
	cid := binary.LittleEndian.Uint16(buf[2:])
	var f Frame
	switch ChannelID(cid) {
	case ChannelIDConnectionless:
		f = &GFrame{}
	default:
		// TODO: handle other channel ids
		f = &BFrame{}
	}
	if f == nil {
		return nil, errors.New("unknown frame type")
	}
	return f, f.Unmarshal(buf)
}

// BFrame is defined in Vol 3, Part A, Section 3.1 of the Bluetooth Core Specification.
type BFrame struct {
	ChannelID
	Payload []byte
}

func (f *BFrame) Marshal() ([]byte, error) {
	if len(f.Payload) > math.MaxUint16 {
		return nil, errors.New("payload too large")
	}
	buf := make([]byte, 4+len(f.Payload))
	binary.LittleEndian.PutUint16(buf[0:], uint16(len(f.Payload)))
	binary.LittleEndian.PutUint16(buf[2:], uint16(f.ChannelID))
	copy(buf[4:], f.Payload)
	return buf, nil
}

func (f *BFrame) Unmarshal(buf []byte) error {
	if len(buf) < 4 || uint16(len(buf)-4) != binary.LittleEndian.Uint16(buf[0:]) {
		return io.ErrShortBuffer
	}
	f.ChannelID = ChannelID(binary.LittleEndian.Uint16(buf[2:]))
	f.Payload = buf[4:]
	return nil
}

// GFrame is defined in Vol 3, Part A, Section 3.2 of the Bluetooth Core Specification.
type GFrame struct {
	PSM     uint16
	Payload []byte
}

func (f *GFrame) Marshal() ([]byte, error) {
	if len(f.Payload) > math.MaxUint16-2 {
		return nil, errors.New("payload too large")
	}
	buf := make([]byte, 6+len(f.Payload))
	binary.LittleEndian.PutUint16(buf[0:], uint16(len(f.Payload)+2))
	binary.LittleEndian.PutUint16(buf[2:], uint16(ChannelIDConnectionless))
	binary.LittleEndian.PutUint16(buf[4:], f.PSM)
	copy(buf[6:], f.Payload)
	return buf, nil
}

func (f *GFrame) Unmarshal(buf []byte) error {
	if len(buf) < 6 {
		return io.ErrShortBuffer
	}
	if uint16(len(buf)-6) != binary.LittleEndian.Uint16(buf[0:]) {
		return io.ErrShortBuffer
	}
	if binary.LittleEndian.Uint16(buf[2:]) != uint16(ChannelIDConnectionless) {
		return errors.New("incorrect channel id")
	}
	f.PSM = binary.LittleEndian.Uint16(buf[4:])
	f.Payload = buf[6:]
	return nil
}
