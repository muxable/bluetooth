package att

type Packet interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}
