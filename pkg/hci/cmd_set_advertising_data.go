package hci

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
)

type HCISetAdvertisingDataCommandPacket struct {
	AdvertisingData []DataType
}

func (p *HCISetAdvertisingDataCommandPacket) Marshal() ([]byte, error) {
	var ads []byte
	for _, data := range p.AdvertisingData {
		ad, err := data.Marshal()
		if err != nil {
			return nil, err
		}
		ads = append(ads, ad...)
	}

	if len(ads) > 31 {
		return nil, io.ErrShortWrite
	}

	log.Printf("%x", ads)

	buf := make([]byte, 36)
	buf[0] = byte(PacketTypeCommand)
	binary.LittleEndian.PutUint16(buf[1:], uint16(OpcodeSetAdvertisingData))
	buf[3] = uint8(32)
	buf[4] = uint8(len(ads))
	copy(buf[5:], ads)
	return buf, nil
}

func (p *HCISetAdvertisingDataCommandPacket) Unmarshal(buf []byte) error {
	return errors.New("not implemented")
}

func (p *HCISetAdvertisingDataCommandPacket) Opcode() Opcode {
	return OpcodeSetAdvertisingData
}

func (a *Adapter) SetAdvertisingData(data ...DataType) error {
	buf1, err := a.op(&HCISetAdvertisingDataCommandPacket{AdvertisingData: data})
	if err != nil {
		return err
	}
	if buf1[0] != 0 {
		return errors.New("command failed")
	}
	return nil
}
