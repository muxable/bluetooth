package hci

type OwnAddressType uint8

const (
	OwnAddressTypePublicDeviceAddress         OwnAddressType = 0x00
	OwnAddressTypeRandomDeviceAddress         OwnAddressType = 0x01
	OwnAddressTypeControllerGeneratedOrPublic OwnAddressType = 0x02
	OwnAddressTypeControllerGeneratedOrRandom OwnAddressType = 0x03
)

type PeerAddressType uint8

const (
	PeerAddressTypePublicDeviceAddress PeerAddressType = 0x00
	PeerAddressTypeRandomDeviceAddress PeerAddressType = 0x01
)

type BDAddr [6]byte
