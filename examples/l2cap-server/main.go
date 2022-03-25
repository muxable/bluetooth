package main

import (
	"log"
	"math"

	"github.com/muxable/bluetooth/pkg/hci"
	"github.com/muxable/bluetooth/pkg/l2cap"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	sck, err := hci.NewSocket(0)
	if err != nil {
		panic(err)
	}

	a := hci.NewConn(sck)

	if err := a.Reset(); err != nil {
		panic(err)
	}

	addr, err := a.ReadBDAddr()
	if err != nil {
		panic(err)
	}

	log.Printf("got bdaddr %x", addr)

	if err := a.ClearFilterAcceptList(); err != nil {
		panic(err)
	}

	n, err := a.ReadFilterAcceptListSize()
	if err != nil {
		panic(err)
	}

	log.Printf("got filter accept list size %v", n)

	if err := a.SetEventMask(
		hci.EventMaskDisconnectionCompleteEvent |
			hci.EventMaskEncryptionChangeEvent |
			hci.EventMaskHardwareErrorEvent |
			hci.EventMaskEncryptionKeyRefreshCompleteEvent |
			hci.EventMaskLEMetaEvent); err != nil {
		panic(err)
	}

	if err := a.LESetEventMask(
		hci.LEEventMaskConnectionCompleteEvent |
			hci.LEEventMaskAdvertisingReportEvent |
			hci.LEEventMaskConnectionUpdateCompleteEvent |
			hci.LEEventMaskLongTermKeyRequestEvent |
			hci.LEEventMaskReadRemoteUsedFeaturesCompleteEvent); err != nil {
		panic(err)
	}

	bs, err := a.LEReadBufferSize()
	if err != nil {
		panic(err)
	}

	log.Printf("got buffer size %v", bs)

	ss, err := a.LEReadSupportedStates()
	if err != nil {
		panic(err)
	}

	log.Printf("got supported states %v", ss)

	if err := a.SetAdvertisingData(
		hci.FlagsDataTypeLEGeneralDiscoverableMode|hci.FlagsDataTypeBREDRNotSupported,
		hci.CompleteLocalName("Muxer")); err != nil {
		panic(err)
	}

	if err := a.LESetAdvertisingParameters(&hci.SetAdvertisingParametersRequest{
		AdvertisingIntervalMin: 100,
		AdvertisingIntervalMax: 120,
	}); err != nil {
		panic(err)
	}

	if err := a.LESetAdvertisingEnable(true); err != nil {
		panic(err)
	}

	for {
		conn, err := a.Accept()
		if err != nil {
			panic(err)
		}

		go func() {
			l2capconn := l2cap.NewConn(conn)
			for {
				channel, err := l2capconn.Accept()
				if err != nil {
					panic(err)
				}
				go func() {
					switch channel.PSM {
					case 0x0080:
						channel.Approve(false, math.MaxUint16)
						buf := make([]byte, math.MaxUint16)
						for {
							n, err := channel.Read(buf)
							if err != nil {
								break
							}
							log.Printf("got %d bytes: %x", n, buf[:n])

							if _, err := channel.Write(buf[:n]); err != nil {
								panic(err)
							}
						}
					default:
						channel.Reject(l2cap.LECreditBasedConnectionResultRefusedSPSMNotSupported)
					}
				}()
			}

		}()
	}
}
