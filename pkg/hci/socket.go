package hci

import (
	"fmt"
	"io"
	"math"
	"sync"
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

func ioR(t, nr, size uintptr) uintptr {
	return (2 << 30) | (t << 8) | nr | (size << 16)
}

func ioW(t, nr, size uintptr) uintptr {
	return (1 << 30) | (t << 8) | nr | (size << 16)
}

func ioctl(fd, op, arg uintptr) error {
	if _, _, ep := unix.Syscall(unix.SYS_IOCTL, fd, op, arg); ep != 0 {
		return ep
	}
	return nil
}

const (
	ioctlSize     = 4
	hciMaxDevices = 16
	typHCI        = 72 // 'H'
)

var (
	hciUpDevice      = ioW(typHCI, 201, ioctlSize) // HCIDEVUP
	hciDownDevice    = ioW(typHCI, 202, ioctlSize) // HCIDEVDOWN
	hciResetDevice   = ioW(typHCI, 203, ioctlSize) // HCIDEVRESET
	hciGetDeviceList = ioR(typHCI, 210, ioctlSize) // HCIGETDEVLIST
	hciGetDeviceInfo = ioR(typHCI, 211, ioctlSize) // HCIGETDEVINFO
)

type devListRequest struct {
	devNum     uint16
	devRequest [hciMaxDevices]struct {
		id  uint16
		opt uint32
	}
}

// Socket implements a HCI User Channel as ReadWriteCloser.
type Socket struct {
	fd     int
	closed chan struct{}
	rmu    sync.Mutex
	wmu    sync.Mutex
}

// NewSocket returns a HCI User Channel of specified device id.
// If id is -1, the first available HCI device is returned.
func NewSocket(id int) (*Socket, error) {
	var err error
	// Create RAW HCI Socket.
	fd, err := unix.Socket(unix.AF_BLUETOOTH, unix.SOCK_RAW, unix.BTPROTO_HCI)
	if err != nil {
		return nil, err
	}

	if id != -1 {
		return open(fd, id)
	}

	req := devListRequest{devNum: hciMaxDevices}
	if err = ioctl(uintptr(fd), hciGetDeviceList, uintptr(unsafe.Pointer(&req))); err != nil {
		return nil, err
	}
	var msg string
	for id := 0; id < int(req.devNum); id++ {
		s, err := open(fd, id)
		if err == nil {
			return s, nil
		}
		msg = msg + fmt.Sprintf("(hci%d: %s)", id, err)
	}
	return nil, fmt.Errorf("no devices available: %s", msg)
}

func open(fd, id int) (*Socket, error) {
	// Reset the device in case previous session didn't cleanup properly.
	if err := ioctl(uintptr(fd), hciDownDevice, uintptr(id)); err != nil {
		return nil, err
	}
	if err := ioctl(uintptr(fd), hciUpDevice, uintptr(id)); err != nil {
		return nil, err
	}

	// HCI User Channel requires exclusive access to the device.
	// The device has to be down at the time of binding.
	if err := ioctl(uintptr(fd), hciDownDevice, uintptr(id)); err != nil {
		return nil, err
	}

	// Bind the RAW socket to HCI User Channel
	sa := unix.SockaddrHCI{Dev: uint16(id), Channel: unix.HCI_CHANNEL_USER}
	if err := unix.Bind(fd, &sa); err != nil {
		return nil, err
	}

	// poll for 20ms to see if any data becomes available, then clear it
	pfds := []unix.PollFd{unix.PollFd{Fd: int32(fd), Events: unix.POLLIN}}
	unix.Poll(pfds, 20)
	if pfds[0].Revents&unix.POLLIN > 0 {
		b := make([]byte, 100)
		unix.Read(fd, b)
	}

	return &Socket{fd: fd, closed: make(chan struct{})}, nil
}

func (s *Socket) Read(p []byte) (int, error) {
	select {
	case <-s.closed:
		return 0, io.EOF
	default:
	}
	s.rmu.Lock()
	defer s.rmu.Unlock()
	return unix.Read(s.fd, p)
}

func (s *Socket) Write(p []byte) (int, error) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	return unix.Write(s.fd, p)
}

func (s *Socket) ReadPacket() (Packet, error) {
	buf := make([]byte, math.MaxUint16)
	n, err := s.Read(buf)
	if err != nil {
		return nil, err
	}
	zap.L().Debug("bluetooth reading", zap.String("packet", fmt.Sprintf("%x", buf[:n])))
	return Unmarshal(buf[:n])
}

func (s *Socket) WritePacket(p Packet) error {
	buf, err := p.Marshal()
	if err != nil {
		return err
	}
	zap.L().Debug("bluetooth writing", zap.String("packet", fmt.Sprintf("%x", buf)))
	_, err = s.Write(buf)
	return err
}

func (s *Socket) Close() error {
	close(s.closed)
	s.Write([]byte{0x01, 0x09, 0x10, 0x00})
	s.rmu.Lock()
	defer s.rmu.Unlock()
	return unix.Close(s.fd)
}
