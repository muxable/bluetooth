package att

type Opcode uint8

// Vol 3, Part F, Section 3.4.8 of the Bluetooth Core Specification
const (
	OpcodeErrorResponse                   Opcode = 0x01
	OpcodeExchangeMTURequest              Opcode = 0x02
	OpcodeExchangeMTUResponse             Opcode = 0x03
	OpcodeFindInformationRequest          Opcode = 0x04
	OpcodeFindInformationResponse         Opcode = 0x05
	OpcodeFindByTypeValueRequest          Opcode = 0x06
	OpcodeFindByTypeValueResponse         Opcode = 0x07
	OpcodeReadByTypeRequest               Opcode = 0x08
	OpcodeReadByTypeResponse              Opcode = 0x09
	OpcodeReadRequest                     Opcode = 0x0A
	OpcodeReadResponse                    Opcode = 0x0B
	OpcodeReadBlobRequest                 Opcode = 0x0C
	OpcodeReadBlobResponse                Opcode = 0x0D
	OpcodeReadMultipleRequest             Opcode = 0x0E
	OpcodeReadMultipleResponse            Opcode = 0x0F
	OpcodeReadByGroupTypeRequest          Opcode = 0x10
	OpcodeReadByGroupTypeResponse         Opcode = 0x11
	OpcodeWriteRequest                    Opcode = 0x12
	OpcodeWriteResponse                   Opcode = 0x13
	OpcodeWriteCommand                    Opcode = 0x52
	OpcodePrepareWriteRequest             Opcode = 0x16
	OpcodePrepareWriteResponse            Opcode = 0x17
	OpcodeExecuteWriteRequest             Opcode = 0x18
	OpcodeExecuteWriteResponse            Opcode = 0x19
	OpcodeReadMultipleVariableRequest     Opcode = 0x20
	OpcodeReadMultipleVariableResponse    Opcode = 0x21
	OpcodeMultipleHandleValueNotification Opcode = 0x23
	OpcodeHandleValueNotification         Opcode = 0x1B
	OpcodeHandleValueIndication           Opcode = 0x1D
	OpcodeHandleValueConfirmation         Opcode = 0x1E
	OpcodeSignedWriteCommand              Opcode = 0xD2
)
