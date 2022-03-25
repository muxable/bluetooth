package l2cap

type Opcode uint8

const (
	OpcodeCommandRejectResponse             Opcode = 0x01
	OpcodeConnectionRequest                 Opcode = 0x02
	OpcodeConnectionResponse                Opcode = 0x03
	OpcodeConfigurationRequest              Opcode = 0x04
	OpcodeConfigurationResponse             Opcode = 0x05
	OpcodeDisconnectionRequest              Opcode = 0x06
	OpcodeDisconnectionResponse             Opcode = 0x07
	OpcodeEchoRequest                       Opcode = 0x08
	OpcodeEchoResponse                      Opcode = 0x09
	OpcodeInformationRequest                Opcode = 0x0A
	OpcodeInformationResponse               Opcode = 0x0B
	OpcodeConnectionParameterUpdateRequest  Opcode = 0x12
	OpcodeConnectionParameterUpdateResponse Opcode = 0x13
	OpcodeLECreditBasedConnectionRequest    Opcode = 0x14
	OpcodeLECreditBasedConnectionResponse   Opcode = 0x15
	OpcodeFlowControlCreditIND              Opcode = 0x16
	OpcodeCreditBasedConnectionRequest      Opcode = 0x17
	OpcodeCreditBasedConnectionResponse     Opcode = 0x18
	OpcodeCreditBasedReconfigureRequest     Opcode = 0x19
	OpcodeCreditBasedReconfigureResponse    Opcode = 0x1A
)

// Section 2.1
type ChannelID uint16

const (
	ChannelIDSignallingACLU          ChannelID = 0x0001
	ChannelIDConnectionless          ChannelID = 0x0002
	ChannelIDAttributeProtocol       ChannelID = 0x0004
	ChannelIDSignallingLEU           ChannelID = 0x0005
	ChannelIDSecurityManagerProtocol ChannelID = 0x0006
	ChannelIDBREDRSecurityManager    ChannelID = 0x0007
)
