package hci

import (
	"encoding/binary"
	"errors"
	"io"
)

type AdvertisingType uint8

const (
	AdvertisingTypeConnectableAndScannableUndirectedAdvertising AdvertisingType = 0x00
	AdvertisingTypeConnectableHighDutyCycleDirectedAdvertising  AdvertisingType = 0x01
	AdvertisingTypeScannableUndirectedAdvertising               AdvertisingType = 0x02
	AdvertisingTypeNonConnectableUndirectedAdvertising          AdvertisingType = 0x03
	AdvertisingTypeConnectableLowDutyCycleDirectedAdvertising   AdvertisingType = 0x04
)

type AdvertisingChannelMap uint8

const (
	AdvertisingChannelMapChannel37 AdvertisingChannelMap = 0x01
	AdvertisingChannelMapChannel38 AdvertisingChannelMap = 0x02
	AdvertisingChannelMapChannel39 AdvertisingChannelMap = 0x04

	AdvertisingChannelMapDefault AdvertisingChannelMap = 0x07
)

type AdvertisingFilterPolicy uint8

const (
	AdvertisingFilterPolicyProcessScanAndConnectionRequestsFromAllDevices                       AdvertisingFilterPolicy = 0x00
	AdvertisingFilterPolicyProcessConnectionRequestsFromAllDevicesAndScanRequestsFromFilterList AdvertisingFilterPolicy = 0x01
	AdvertisingFilterPolicyProcessScanRequestsFromAllDevicesAndConnectionRequestsFromFilterList AdvertisingFilterPolicy = 0x02
	AdvertisingFilterPolicyProcessScanAndConnectionRequestsFromFilterList                       AdvertisingFilterPolicy = 0x03
)

type HCILESetAdvertisingParametersCommandPacket struct {
	AdvertisingIntervalMin  uint16
	AdvertisingIntervalMax  uint16
	AdvertisingType         AdvertisingType
	OwnAddressType          OwnAddressType
	PeerAddressType         PeerAddressType
	PeerAddress             BDAddr
	AdvertisingChannelMap   AdvertisingChannelMap
	AdvertisingFilterPolicy AdvertisingFilterPolicy
}

func (p *HCILESetAdvertisingParametersCommandPacket) Marshal() ([]byte, error) {
	buf := make([]byte, 19)
	buf[0] = byte(PacketTypeCommand)
	binary.LittleEndian.PutUint16(buf[1:], uint16(OpcodeLESetAdvertisingParameters))
	buf[3] = 15
	binary.LittleEndian.PutUint16(buf[4:], p.AdvertisingIntervalMin)
	binary.LittleEndian.PutUint16(buf[6:], p.AdvertisingIntervalMax)
	buf[8] = byte(p.AdvertisingType)
	buf[9] = byte(p.OwnAddressType)
	buf[10] = byte(p.PeerAddressType)
	copy(buf[11:], p.PeerAddress[:])
	buf[17] = byte(p.AdvertisingChannelMap)
	buf[18] = byte(p.AdvertisingFilterPolicy)
	return buf, nil
}

func (p *HCILESetAdvertisingParametersCommandPacket) Unmarshal(buf []byte) error {
	if len(buf) < 19 {
		return io.ErrUnexpectedEOF
	}
	p.AdvertisingIntervalMin = binary.LittleEndian.Uint16(buf[4:])
	p.AdvertisingIntervalMax = binary.LittleEndian.Uint16(buf[6:])
	p.AdvertisingType = AdvertisingType(buf[8])
	p.OwnAddressType = OwnAddressType(buf[9])
	p.PeerAddressType = PeerAddressType(buf[10])
	copy(p.PeerAddress[:], buf[11:17])
	p.AdvertisingChannelMap = AdvertisingChannelMap(buf[17])
	p.AdvertisingFilterPolicy = AdvertisingFilterPolicy(buf[18])
	return nil
}

func (p *HCILESetAdvertisingParametersCommandPacket) Opcode() Opcode {
	return OpcodeLESetAdvertisingParameters
}

type SetAdvertisingParametersRequest struct {
	AdvertisingIntervalMin  uint16
	AdvertisingIntervalMax  uint16
	AdvertisingType         AdvertisingType
	OwnAddressType          OwnAddressType
	PeerAddressType         PeerAddressType
	PeerAddress             BDAddr
	AdvertisingChannelMap   AdvertisingChannelMap
	AdvertisingFilterPolicy AdvertisingFilterPolicy
}

func (a *Adapter) LESetAdvertisingParameters(request *SetAdvertisingParametersRequest) error {
	if request.AdvertisingIntervalMin == 0 {
		request.AdvertisingIntervalMin = 0x0800
	}
	if request.AdvertisingIntervalMin < 0x0020 || request.AdvertisingIntervalMin > 0x4000 {
		return errors.New("invalid advertising interval min")
	}
	if request.AdvertisingIntervalMax == 0 {
		request.AdvertisingIntervalMax = 0x0800
	}
	if request.AdvertisingIntervalMax < 0x0020 || request.AdvertisingIntervalMax > 0x4000 {
		return errors.New("invalid advertising interval max")
	}
	if request.AdvertisingChannelMap == 0 {
		request.AdvertisingChannelMap = AdvertisingChannelMapDefault
	}

	buf, err := a.op(&HCILESetAdvertisingParametersCommandPacket{
		AdvertisingIntervalMin:  request.AdvertisingIntervalMin,
		AdvertisingIntervalMax:  request.AdvertisingIntervalMax,
		AdvertisingType:         request.AdvertisingType,
		OwnAddressType:          request.OwnAddressType,
		PeerAddressType:         request.PeerAddressType,
		PeerAddress:             request.PeerAddress,
		AdvertisingChannelMap:   request.AdvertisingChannelMap,
		AdvertisingFilterPolicy: request.AdvertisingFilterPolicy,
	})
	if err != nil {
		return err
	}
	if buf[0] != 0 {
		return errors.New("command failed")
	}
	return nil
}
