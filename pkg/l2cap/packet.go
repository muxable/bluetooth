package l2cap

import (
	"encoding/binary"
	"errors"
	"io"
)

var identifier uint8

func NextIdentifier() uint8 {
	defer func() { identifier++ }()
	return identifier
}

type SignallingPacket interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

func UnmarshalSignallingPacket(buf []byte) (SignallingPacket, error) {
	if len(buf) < 1 {
		return nil, io.ErrShortBuffer
	}
	var p SignallingPacket
	switch Opcode(buf[0]) {
	case OpcodeCommandRejectResponse:
		p = &CommandRejectResponsePacket{}
	case OpcodeConnectionRequest:
		p = &ConnectionRequestPacket{}
	case OpcodeConnectionResponse:
		p = &ConnectionResponsePacket{}
	case OpcodeConfigurationRequest:
		return nil, errors.New("not implemented")
	case OpcodeConfigurationResponse:
		return nil, errors.New("not implemented")
	case OpcodeDisconnectionRequest:
		p = &DisconnectionRequestPacket{}
	case OpcodeDisconnectionResponse:
		p = &DisconnectionResponsePacket{}
	case OpcodeEchoRequest:
		p = &EchoRequestPacket{}
	case OpcodeEchoResponse:
		p = &EchoResponsePacket{}
	case OpcodeInformationRequest:
		p = &InformationRequestPacket{}
	case OpcodeInformationResponse:
		p = &InformationResponsePacket{}
	case OpcodeConnectionParameterUpdateRequest:
		p = &ConnectionParameterUpdateRequestPacket{}
	case OpcodeConnectionParameterUpdateResponse:
		p = &ConnectionParameterUpdateResponsePacket{}
	case OpcodeLECreditBasedConnectionRequest:
		p = &LECreditBasedConnectionRequestPacket{}
	case OpcodeLECreditBasedConnectionResponse:
		p = &LECreditBasedConnectionResponsePacket{}
	case OpcodeFlowControlCreditIND:
		p = &FlowControlCreditIndicationPacket{}
	case OpcodeCreditBasedConnectionRequest:
		p = &CreditBasedConnectionRequestPacket{}
	case OpcodeCreditBasedConnectionResponse:
		p = &CreditBasedConnectionResponsePacket{}
	case OpcodeCreditBasedReconfigureRequest:
		return nil, errors.New("not implemented")
	case OpcodeCreditBasedReconfigureResponse:
		return nil, errors.New("not implemented")
	}
	if p == nil {
		return nil, errors.New("invalid opcode")
	}
	return p, p.Unmarshal(buf)
}

type CommandRejectReason uint16

const (
	CommandRejectReasonCommandNotUnderstood CommandRejectReason = 0x0000
	CommandRejectReasonSignalingMTUExceeded CommandRejectReason = 0x0001
	CommandRejectReasonInvalidCIDInRequest  CommandRejectReason = 0x0002
)

type CommandRejectResponsePacket struct {
	CommandRejectReason
	Identifier uint8
	ReasonData []byte
}

func (p *CommandRejectResponsePacket) Marshal() ([]byte, error) {
	b := make([]byte, 6+len(p.ReasonData))
	b[0] = byte(OpcodeCommandRejectResponse)
	b[1] = byte(p.Identifier)
	binary.LittleEndian.PutUint16(b[2:], uint16(len(p.ReasonData)+2))
	binary.LittleEndian.PutUint16(b[4:], uint16(p.CommandRejectReason))
	copy(b[6:], p.ReasonData)
	return b, nil
}

func (p *CommandRejectResponsePacket) Unmarshal(buf []byte) error {
	if len(buf) < 6 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeCommandRejectResponse) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	p.CommandRejectReason = CommandRejectReason(binary.LittleEndian.Uint16(buf[4:]))
	p.ReasonData = buf[6:]
	if len(p.ReasonData) != int(binary.LittleEndian.Uint16(buf[2:4]))-2 {
		return io.ErrShortBuffer
	}
	return nil
}

type ConnectionRequestPacket struct {
	Identifier uint8
	PSM        uint16
	SourceCID  uint16
}

func (p *ConnectionRequestPacket) Marshal() ([]byte, error) {
	b := make([]byte, 8)
	b[0] = byte(OpcodeConnectionRequest)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], 4)
	binary.LittleEndian.PutUint16(b[4:], p.PSM)
	binary.LittleEndian.PutUint16(b[6:], p.SourceCID)
	return b, nil
}

func (p *ConnectionRequestPacket) Unmarshal(buf []byte) error {
	if len(buf) < 8 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeConnectionRequest) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != 4 {
		return errors.New("invalid length")
	}
	p.PSM = binary.LittleEndian.Uint16(buf[4:])
	p.SourceCID = binary.LittleEndian.Uint16(buf[6:])
	return nil
}

type ConnectionResponseResult uint16

const (
	ConnectionResponseResultSuccessfulConnection             ConnectionResponseResult = 0x0000
	ConnectionResponseResultPending                          ConnectionResponseResult = 0x0001
	ConnectionResponseResultRefusedPSMNotSUpported           ConnectionResponseResult = 0x0002
	ConnectionResponseResultRefusedSecurityBlock             ConnectionResponseResult = 0x0003
	ConnectionResponseResultRefusedNoResourcesAvailable      ConnectionResponseResult = 0x0004
	ConnectionResponseResultRefusedInvalidSourceCID          ConnectionResponseResult = 0x0006
	ConnectionResponseResultRefusedSourceCIDAlreadyAllocated ConnectionResponseResult = 0x0007
)

type ConnectionResponseStatus uint16

const (
	ConnectionResponseStatusNoFurtherInformationAvailable ConnectionResponseStatus = 0x0000
	ConnectionResponseStatusAuthenticationPending         ConnectionResponseStatus = 0x0001
	ConnectionResponseStatusAuthorizationPending          ConnectionResponseStatus = 0x0002
)

type ConnectionResponsePacket struct {
	Identifier     uint8
	DestinationCID uint16
	SourceCID      uint16
	Result         ConnectionResponseResult
	Status         ConnectionResponseStatus
}

func (p *ConnectionResponsePacket) Marshal() ([]byte, error) {
	b := make([]byte, 12)
	b[0] = byte(OpcodeConnectionResponse)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], 8)
	binary.LittleEndian.PutUint16(b[4:], p.DestinationCID)
	binary.LittleEndian.PutUint16(b[6:], p.SourceCID)
	binary.LittleEndian.PutUint16(b[8:], uint16(p.Result))
	binary.LittleEndian.PutUint16(b[10:], uint16(p.Status))
	return b, nil
}

func (p *ConnectionResponsePacket) Unmarshal(buf []byte) error {
	if len(buf) < 12 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeConnectionResponse) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != 8 {
		return errors.New("invalid length")
	}
	p.DestinationCID = binary.LittleEndian.Uint16(buf[4:])
	p.SourceCID = binary.LittleEndian.Uint16(buf[6:])
	p.Result = ConnectionResponseResult(binary.LittleEndian.Uint16(buf[8:]))
	p.Status = ConnectionResponseStatus(binary.LittleEndian.Uint16(buf[10:]))
	return nil
}

type DisconnectionRequestPacket struct {
	Identifier     uint8
	DestinationCID ChannelID
	SourceCID      ChannelID
}

func (p *DisconnectionRequestPacket) Marshal() ([]byte, error) {
	b := make([]byte, 8)
	b[0] = byte(OpcodeDisconnectionRequest)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], 4)
	binary.LittleEndian.PutUint16(b[4:], uint16(p.DestinationCID))
	binary.LittleEndian.PutUint16(b[6:], uint16(p.SourceCID))
	return b, nil
}

func (p *DisconnectionRequestPacket) Unmarshal(buf []byte) error {
	if len(buf) < 8 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeDisconnectionRequest) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != 4 {
		return errors.New("invalid length")
	}
	p.DestinationCID = ChannelID(binary.LittleEndian.Uint16(buf[4:]))
	p.SourceCID = ChannelID(binary.LittleEndian.Uint16(buf[6:]))
	return nil
}

type DisconnectionResponsePacket struct {
	Identifier     uint8
	DestinationCID ChannelID
	SourceCID      ChannelID
}

func (p *DisconnectionResponsePacket) Marshal() ([]byte, error) {
	b := make([]byte, 8)
	b[0] = byte(OpcodeDisconnectionResponse)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], 4)
	binary.LittleEndian.PutUint16(b[4:], uint16(p.DestinationCID))
	binary.LittleEndian.PutUint16(b[6:], uint16(p.SourceCID))
	return b, nil
}

func (p *DisconnectionResponsePacket) Unmarshal(buf []byte) error {
	if len(buf) < 8 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeDisconnectionResponse) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != 4 {
		return errors.New("invalid length")
	}
	p.DestinationCID = ChannelID(binary.LittleEndian.Uint16(buf[4:]))
	p.SourceCID = ChannelID(binary.LittleEndian.Uint16(buf[6:]))
	return nil
}

type EchoRequestPacket struct {
	Identifier uint8
	EchoData   []byte
}

func (p *EchoRequestPacket) Marshal() ([]byte, error) {
	b := make([]byte, 2+len(p.EchoData))
	b[0] = byte(OpcodeEchoRequest)
	b[1] = p.Identifier
	copy(b[2:], p.EchoData)
	return b, nil
}

func (p *EchoRequestPacket) Unmarshal(buf []byte) error {
	if len(buf) < 2 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeEchoRequest) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != uint16(len(buf)-2) {
		return errors.New("invalid length")
	}
	p.EchoData = buf[2:]
	return nil
}

type EchoResponsePacket struct {
	Identifier uint8
	EchoData   []byte
}

func (p *EchoResponsePacket) Marshal() ([]byte, error) {
	b := make([]byte, 2+len(p.EchoData))
	b[0] = byte(OpcodeEchoResponse)
	b[1] = p.Identifier
	copy(b[2:], p.EchoData)
	return b, nil
}

func (p *EchoResponsePacket) Unmarshal(buf []byte) error {
	if len(buf) < 2 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeEchoResponse) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != uint16(len(buf)-2) {
		return errors.New("invalid length")
	}
	p.EchoData = buf[2:]
	return nil
}

type InfoType uint16

const (
	InfoTypeConnectionlessMTU         InfoType = 0x0001
	InfoTypeExtendedFeaturesSupported InfoType = 0x0002
	InfoTypeFixedChannelsSupported    InfoType = 0x0003
)

type InformationRequestPacket struct {
	Identifier uint8
	InfoType
}

func (p *InformationRequestPacket) Marshal() ([]byte, error) {
	b := make([]byte, 4)
	b[0] = byte(OpcodeInformationRequest)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], 2)
	binary.LittleEndian.PutUint16(b[4:], uint16(p.InfoType))
	return b, nil
}

func (p *InformationRequestPacket) Unmarshal(buf []byte) error {
	if len(buf) < 4 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeInformationRequest) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != 2 {
		return errors.New("invalid length")
	}
	p.InfoType = InfoType(binary.LittleEndian.Uint16(buf[4:]))
	return nil
}

type InfoTypeResult uint16

const (
	InfoTypeResultSuccess      InfoTypeResult = 0x0000
	InfoTypeResultNotSupported InfoTypeResult = 0x0001
)

type InformationResponsePacket struct {
	Identifier uint8
	InfoType
	Result InfoTypeResult
	Info   []byte
}

func (p *InformationResponsePacket) Marshal() ([]byte, error) {
	b := make([]byte, 8+len(p.Info))
	b[0] = byte(OpcodeInformationResponse)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], uint16(len(p.Info)+4))
	binary.LittleEndian.PutUint16(b[4:], uint16(p.InfoType))
	binary.LittleEndian.PutUint16(b[6:], uint16(p.Result))
	copy(b[8:], p.Info)
	return b, nil
}

func (p *InformationResponsePacket) Unmarshal(buf []byte) error {
	if len(buf) < 8 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeInformationResponse) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != uint16(len(buf)-4) {
		return errors.New("invalid length")
	}
	p.InfoType = InfoType(binary.LittleEndian.Uint16(buf[4:]))
	p.Result = InfoTypeResult(binary.LittleEndian.Uint16(buf[6:]))
	p.Info = buf[8:]
	return nil
}

// TODO: implement parsing of info field.

type ConnectionParameterUpdateRequestPacket struct {
	Identifier  uint8
	IntervalMin uint16
	IntervalMax uint16
	Latency     uint16
	Timeout     uint16
}

func (p *ConnectionParameterUpdateRequestPacket) Marshal() ([]byte, error) {
	b := make([]byte, 12)
	b[0] = byte(OpcodeConnectionParameterUpdateRequest)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], 8)
	binary.LittleEndian.PutUint16(b[4:], p.IntervalMin)
	binary.LittleEndian.PutUint16(b[6:], p.IntervalMax)
	binary.LittleEndian.PutUint16(b[8:], p.Latency)
	binary.LittleEndian.PutUint16(b[10:], p.Timeout)
	return b, nil
}

func (p *ConnectionParameterUpdateRequestPacket) Unmarshal(buf []byte) error {
	if len(buf) < 12 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeConnectionParameterUpdateRequest) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != 8 {
		return errors.New("invalid length")
	}
	p.IntervalMin = binary.LittleEndian.Uint16(buf[4:])
	p.IntervalMax = binary.LittleEndian.Uint16(buf[6:])
	p.Latency = binary.LittleEndian.Uint16(buf[8:])
	p.Timeout = binary.LittleEndian.Uint16(buf[10:])
	return nil
}

type ConnectionParameterUpdateResult uint16

const (
	ConnectionParameterUpdateResultAccepted ConnectionParameterUpdateResult = 0x0000
	ConnectionParameterUpdateResultRejected ConnectionParameterUpdateResult = 0x0001
)

type ConnectionParameterUpdateResponsePacket struct {
	Identifier uint8
	Result     ConnectionParameterUpdateResult
}

func (p *ConnectionParameterUpdateResponsePacket) Marshal() ([]byte, error) {
	b := make([]byte, 6)
	b[0] = byte(OpcodeConnectionParameterUpdateResponse)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], 2)
	binary.LittleEndian.PutUint16(b[4:], uint16(p.Result))
	return b, nil
}

func (p *ConnectionParameterUpdateResponsePacket) Unmarshal(buf []byte) error {
	if len(buf) < 6 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeConnectionParameterUpdateResponse) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != 2 {
		return errors.New("invalid length")
	}
	p.Result = ConnectionParameterUpdateResult(binary.LittleEndian.Uint16(buf[4:]))
	return nil
}

type LECreditBasedConnectionRequestPacket struct {
	Identifier     uint8
	SPSM           uint16
	SourceCID      ChannelID
	MTU            uint16
	MPS            uint16
	InitialCredits uint16
}

func (p *LECreditBasedConnectionRequestPacket) Marshal() ([]byte, error) {
	b := make([]byte, 14)
	b[0] = byte(OpcodeLECreditBasedConnectionRequest)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], 10)
	binary.LittleEndian.PutUint16(b[4:], p.SPSM)
	binary.LittleEndian.PutUint16(b[6:], uint16(p.SourceCID))
	binary.LittleEndian.PutUint16(b[8:], p.MTU)
	binary.LittleEndian.PutUint16(b[10:], p.MPS)
	binary.LittleEndian.PutUint16(b[12:], p.InitialCredits)
	return b, nil
}

func (p *LECreditBasedConnectionRequestPacket) Unmarshal(buf []byte) error {
	if len(buf) < 14 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeLECreditBasedConnectionRequest) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != 10 {
		return errors.New("invalid length")
	}
	p.SPSM = binary.LittleEndian.Uint16(buf[4:])
	p.SourceCID = ChannelID(binary.LittleEndian.Uint16(buf[6:]))
	p.MTU = binary.LittleEndian.Uint16(buf[8:])
	p.MPS = binary.LittleEndian.Uint16(buf[10:])
	p.InitialCredits = binary.LittleEndian.Uint16(buf[12:])
	return nil
}

type LECreditBasedConnectionResult uint16

const (
	LECreditBasedConnectionResultSuccessful                        LECreditBasedConnectionResult = 0x0000
	LECreditBasedConnectionResultRefusedSPSMNotSupported           LECreditBasedConnectionResult = 0x0002
	LECreditBasedConnectionResultRefusedNoResourcesAvailable       LECreditBasedConnectionResult = 0x0004
	LECreditBasedConnectionResultRefusedInsufficientAuthentication LECreditBasedConnectionResult = 0x0005
	LECreditBasedConnectionResultRefusedInsufficientAuthorization  LECreditBasedConnectionResult = 0x0006
	LECreditBasedConnectionResultRefusedEncryptionKeySizeTooShort  LECreditBasedConnectionResult = 0x0007
	LECreditBasedConnectionResultRefusedInsufficientEncryption     LECreditBasedConnectionResult = 0x0008
	LECreditBasedConnectionResultRefusedInvalidSourceCID           LECreditBasedConnectionResult = 0x0009
	LECreditBasedConnectionResultRefusedSourceCIDAlreadyAllocated  LECreditBasedConnectionResult = 0x000A
	LECreditBasedConnectionResultRefusedUnacceptableParameters     LECreditBasedConnectionResult = 0x000B
)

type LECreditBasedConnectionResponsePacket struct {
	Identifier     uint8
	DestinationCID ChannelID
	MTU            uint16
	MPS            uint16
	InitialCredits uint16
	Result         LECreditBasedConnectionResult
}

func (p *LECreditBasedConnectionResponsePacket) Marshal() ([]byte, error) {
	b := make([]byte, 14)
	b[0] = byte(OpcodeLECreditBasedConnectionResponse)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], 10)
	binary.LittleEndian.PutUint16(b[4:], uint16(p.DestinationCID))
	binary.LittleEndian.PutUint16(b[6:], p.MTU)
	binary.LittleEndian.PutUint16(b[8:], p.MPS)
	binary.LittleEndian.PutUint16(b[10:], p.InitialCredits)
	binary.LittleEndian.PutUint16(b[12:], uint16(p.Result))
	return b, nil
}

func (p *LECreditBasedConnectionResponsePacket) Unmarshal(buf []byte) error {
	if len(buf) < 14 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeLECreditBasedConnectionResponse) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != 10 {
		return errors.New("invalid length")
	}
	p.DestinationCID = ChannelID(binary.LittleEndian.Uint16(buf[4:]))
	p.MTU = binary.LittleEndian.Uint16(buf[6:])
	p.MPS = binary.LittleEndian.Uint16(buf[8:])
	p.InitialCredits = binary.LittleEndian.Uint16(buf[10:])
	p.Result = LECreditBasedConnectionResult(binary.LittleEndian.Uint16(buf[12:]))
	return nil
}

type FlowControlCreditIndicationPacket struct {
	Identifier uint8
	CID        ChannelID
	Credits    uint16
}

func (p *FlowControlCreditIndicationPacket) Marshal() ([]byte, error) {
	b := make([]byte, 8)
	b[0] = byte(OpcodeFlowControlCreditIND)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], 4)
	binary.LittleEndian.PutUint16(b[4:], uint16(p.CID))
	binary.LittleEndian.PutUint16(b[6:], p.Credits)
	return b, nil
}

func (p *FlowControlCreditIndicationPacket) Unmarshal(buf []byte) error {
	if len(buf) < 8 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeFlowControlCreditIND) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != 4 {
		return errors.New("invalid length")
	}
	p.CID = ChannelID(binary.LittleEndian.Uint16(buf[4:]))
	p.Credits = binary.LittleEndian.Uint16(buf[6:])
	return nil
}

type CreditBasedConnectionRequestPacket struct {
	Identifier     uint8
	SPSM           uint16
	MTU            uint16
	MPS            uint16
	InitialCredits uint16
	SourceCIDs     []uint16
}

func (p *CreditBasedConnectionRequestPacket) Marshal() ([]byte, error) {
	b := make([]byte, 12+len(p.SourceCIDs)*2)
	b[0] = byte(OpcodeCreditBasedConnectionRequest)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], uint16(len(p.SourceCIDs)*2+10))
	binary.LittleEndian.PutUint16(b[4:], p.SPSM)
	binary.LittleEndian.PutUint16(b[6:], p.MTU)
	binary.LittleEndian.PutUint16(b[8:], p.MPS)
	binary.LittleEndian.PutUint16(b[10:], p.InitialCredits)
	for i, cid := range p.SourceCIDs {
		binary.LittleEndian.PutUint16(b[12+i*2:], cid)
	}
	return b, nil
}

func (p *CreditBasedConnectionRequestPacket) Unmarshal(buf []byte) error {
	if len(buf) < 12 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeCreditBasedConnectionRequest) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != uint16(len(buf[4:])) {
		return errors.New("invalid length")
	}
	p.SPSM = binary.LittleEndian.Uint16(buf[4:])
	p.MTU = binary.LittleEndian.Uint16(buf[6:])
	p.MPS = binary.LittleEndian.Uint16(buf[8:])
	p.InitialCredits = binary.LittleEndian.Uint16(buf[10:])
	p.SourceCIDs = make([]uint16, (len(buf)-12)/2)
	for i := 0; i < len(p.SourceCIDs); i++ {
		p.SourceCIDs[i] = binary.LittleEndian.Uint16(buf[12+i*2:])
	}
	return nil
}

type CreditBasedConnectionResult uint16

const (
	CreditBasedConnectionResultAllConnectionsSuccessful                             CreditBasedConnectionResult = 0x0000
	CreditBasedConnectionResultAllConnectionsRefusedSPSMNotSupported                CreditBasedConnectionResult = 0x0002
	CreditBasedConnectionResultSomeConnectionsRefusedInsufficientResourcesAvailable CreditBasedConnectionResult = 0x0004
	CreditBasedConnectionResultAllConnectionsRefusedInsufficientAuthentication      CreditBasedConnectionResult = 0x0005
	CreditBasedConnectionResultAllConnectionsRefusedInsufficientAuthorization       CreditBasedConnectionResult = 0x0006
	CreditBasedConnectionResultAllConnectionsRefusedEncryptionKeySizeTooShort       CreditBasedConnectionResult = 0x0007
	CreditBasedConnectionResultAllConnectionsRefusedInsufficientEncryption          CreditBasedConnectionResult = 0x0008
	CreditBasedConnectionResultSomeConnectionsRefusedInvalidSourceCID               CreditBasedConnectionResult = 0x0009
	CreditBasedConnectionResultSomeConnectionsRefusedSourceCIDAlreadyAllocated      CreditBasedConnectionResult = 0x000A
	CreditBasedConnectionResultAllConnectionsRefusedUnacceptableParameters          CreditBasedConnectionResult = 0x000B
	CreditBasedConnectionResultAllConnectionsRefusedInvalidParameters               CreditBasedConnectionResult = 0x000C
	CreditBasedConnectionResultAllConnectionsPendingNoFurtherInformationAvailable   CreditBasedConnectionResult = 0x000D
	CreditBasedConnectionResultAllConnectionsPendingAuthenticationPending           CreditBasedConnectionResult = 0x000E
	CreditBasedConnectionResultAllConnectionsPendingAuthorizationPending            CreditBasedConnectionResult = 0x000F
)

type CreditBasedConnectionResponsePacket struct {
	Identifier      uint8
	MTU             uint16
	MPS             uint16
	InitialCredits  uint16
	Result          CreditBasedConnectionResult
	DestinationCIDs []uint16
}

func (p *CreditBasedConnectionResponsePacket) Marshal() ([]byte, error) {
	b := make([]byte, 12+len(p.DestinationCIDs)*2)
	b[0] = byte(OpcodeCreditBasedConnectionResponse)
	b[1] = p.Identifier
	binary.LittleEndian.PutUint16(b[2:], uint16(len(p.DestinationCIDs)*2+8))
	binary.LittleEndian.PutUint16(b[4:], p.MTU)
	binary.LittleEndian.PutUint16(b[6:], p.MPS)
	binary.LittleEndian.PutUint16(b[8:], p.InitialCredits)
	binary.LittleEndian.PutUint16(b[10:], uint16(p.Result))
	for i, cid := range p.DestinationCIDs {
		binary.LittleEndian.PutUint16(b[12+i*2:], cid)
	}
	return b, nil
}

func (p *CreditBasedConnectionResponsePacket) Unmarshal(buf []byte) error {
	if len(buf) < 12 {
		return io.ErrShortBuffer
	}
	if buf[0] != byte(OpcodeCreditBasedConnectionResponse) {
		return errors.New("invalid opcode")
	}
	p.Identifier = buf[1]
	if binary.LittleEndian.Uint16(buf[2:]) != uint16(len(buf[4:])) {
		return errors.New("invalid length")
	}
	p.MTU = binary.LittleEndian.Uint16(buf[4:])
	p.MPS = binary.LittleEndian.Uint16(buf[6:])
	p.InitialCredits = binary.LittleEndian.Uint16(buf[8:])
	p.Result = CreditBasedConnectionResult(binary.LittleEndian.Uint16(buf[10:]))
	p.DestinationCIDs = make([]uint16, (len(buf)-12)/2)
	for i := 0; i < len(p.DestinationCIDs); i++ {
		p.DestinationCIDs[i] = binary.LittleEndian.Uint16(buf[12+i*2:])
	}
	return nil
}
