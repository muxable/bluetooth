package hci

// Blueooth Core Specification
type DataType interface {
	Marshal() ([]byte, error)
}

type FlagsDataType uint8

const (
	FlagsDataTypeLELimitedDiscoverableMode                           FlagsDataType = (1 << 0)
	FlagsDataTypeLEGeneralDiscoverableMode                           FlagsDataType = (1 << 1)
	FlagsDataTypeBREDRNotSupported                                   FlagsDataType = (1 << 2)
	FlagsDataTypeSimultaneousLEAndBREDRTosameDeviceCapableController FlagsDataType = (1 << 3)
)

func (f FlagsDataType) Marshal() ([]byte, error) {
	return []byte{0x02, 0x01, byte(f)}, nil
}

type CompleteLocalName string

func (l CompleteLocalName) Marshal() ([]byte, error) {
	return append([]byte{byte(len(l) + 1), 0x09}, []byte(l)...), nil
}

type ShortLocalName string

func (l ShortLocalName) Marshal() ([]byte, error) {
	return append([]byte{byte(len(l) + 1), 0x08}, []byte(l)...), nil
}
