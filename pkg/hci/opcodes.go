package hci

// https://software-dl.ti.com/simplelink/esd/simplelink_cc13x2_sdk/1.60.00.29_new/exports/docs/ble5stack/vendor_specific_guide/BLE_Vendor_Specific_HCI_Guide/hci_interface.html

type PacketType uint8

const (
	PacketTypeCommand         PacketType = 0x01
	PacketTypeACLData         PacketType = 0x02
	PacketTypeSynchronousData PacketType = 0x03
	PacketTypeEvent           PacketType = 0x04
	PacketTypeExtendedCommand PacketType = 0x09
)

type Opcode uint16

const (
	OpcodeReset                      Opcode = 0x0C03
	OpcodeReadBDAddr                 Opcode = 0x1009
	OpcodeClearFilterAcceptList      Opcode = 0x2010
	OpcodeReadFilterAcceptListSize   Opcode = 0x200F
	OpcodeSetEventMask               Opcode = 0x0c01
	OpcodeLESetEventMask             Opcode = 0x2001
	OpcodeLEReadBufferSize           Opcode = 0x2002
	OpcodeLEReadSupportedStates      Opcode = 0x201C
	OpcodeSetAdvertisingData         Opcode = 0x2008
	OpcodeLESetAdvertisingParameters Opcode = 0x2006
	OpcodeLESetAdvertisingEnable     Opcode = 0x200A
)

type EventCode uint8

const (
	EventCodeDisconnectionComplete                EventCode = 0x05
	EventCodeEncryptionChange                     EventCode = 0x08
	EventCodeReadRemoteVersionInformationComplete EventCode = 0x0C
	EventCodeCommandComplete                      EventCode = 0x0E
	EventCodeCommandStatus                        EventCode = 0x0F
	EventCodeHardwareError                        EventCode = 0x10
	EventCodeNumberOfCompletedPackets             EventCode = 0x13
	EventCodeDataBufferOverflow                   EventCode = 0x1A
	EventCodeEncryptionKeyRefreshComplete         EventCode = 0x30
	EventCodeAuthenticatedPayloadTimeoutExpired   EventCode = 0x57
	EventCodeLEMeta                               EventCode = 0x3E
)

type LEMetaSubeventCode uint8

const (
	LEMetaSubeventCodeConnectionComplete             LEMetaSubeventCode = 0x01
	LEMetaSubeventCodeAdvertisingReport              LEMetaSubeventCode = 0x02
	LEMetaSubeventCodeConnectionUpdate               LEMetaSubeventCode = 0x03
	LEMetaSubeventCodeReadRemoteUsedFeaturesComplete LEMetaSubeventCode = 0x04
	LEMetaSubeventCodeLongTermKeyRequest             LEMetaSubeventCode = 0x05
	LEMetaSubeventCodeReadLocalP256PublicKeyComplete LEMetaSubeventCode = 0x08
	LEMetaSubeventCodeGenerateDHKeyComplete          LEMetaSubeventCode = 0x09
	LEMetaSubeventCodeEnhancedConnectionComplete     LEMetaSubeventCode = 0x0A
	LEMetaSubeventCodePHYUpdateComplete              LEMetaSubeventCode = 0x0C
	LEMetaSubeventCodeExtendedAdvertisingReport      LEMetaSubeventCode = 0x0D
)
