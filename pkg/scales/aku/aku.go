package aku

import (
	"context"
	"errors"
	"fmt"
	"github.com/mlsorensen/goscale"
	"github.com/mlsorensen/goscale/pkg/scales/aku/comms"
	"log"
	"time"
	"tinygo.org/x/bluetooth"
)

func init() {
	goscale.Register("Varia AKU", New)
}

type AkuScale struct {
	name           string
	address        bluetooth.Address
	disconnectCtx  context.Context
	disconnectFunc context.CancelFunc
	connected      bool

	btDevice   bluetooth.Device
	writeChar  bluetooth.DeviceCharacteristic
	notifyChar bluetooth.DeviceCharacteristic

	weightUpdateChan chan goscale.WeightUpdate
	lastNotified     time.Time
}

// This line is the compile-time check. It will fail to compile if
// *AkuScale ever stops satisfying the goscale.Scale interface.
var _ goscale.Scale = (*AkuScale)(nil)

func New(device *goscale.FoundDevice) goscale.Scale {
	return &AkuScale{
		name:    device.Name,
		address: device.Address,
	}
}

func (a *AkuScale) Connect() (<-chan goscale.WeightUpdate, error) {
	err := goscale.TryEnableAdapter()
	if err != nil {
		return nil, err
	}

	a.weightUpdateChan = make(chan goscale.WeightUpdate, 20)

	a.disconnectCtx, a.disconnectFunc = context.WithCancel(context.Background())

	a.btDevice, err = goscale.BTAdapter.Connect(a.address, bluetooth.ConnectionParams{})

	if err != nil {
		return nil, err
	}

	err = a.setupCharacteristics()
	if err != nil {
		_ = a.Disconnect()
		return nil, err
	}

	log.Println("setting up notifications")
	err = a.setupNotifications()
	if err != nil {
		_ = a.Disconnect()
		return nil, err
	}
	a.lastNotified = time.Now()

	a.connected = true

	// start the connectivity monitor
	go func() {
		for {
			select {
			case <-a.disconnectCtx.Done():
				_ = a.Disconnect()
				return
			default:
				// If we haven't received notifications in a while, disconnect
				if time.Now().After(a.lastNotified.Add(time.Second)) {
					_ = a.Disconnect()
				}
			}
		}
	}()

	return a.weightUpdateChan, nil
}

func (a *AkuScale) Disconnect() error {
	err := a.btDevice.Disconnect()
	if err != nil {
		// are we still connected or not? who knows
		return err
	}
	//TODO: mutex
	if a.weightUpdateChan != nil {
		close(a.weightUpdateChan)
	}
	a.disconnectFunc()
	a.connected = false
	return nil
}

func (a *AkuScale) IsConnected() bool {
	return a.connected
}

func (a *AkuScale) DeviceName() string {
	return a.name
}

func (a *AkuScale) DisplayName() string {
	return "Varia AKU scale"
}

func (a *AkuScale) Tare(blocking bool) error {
	buf := []byte{0xfa, 0x82, 0x01, 0x01}
	xor := buf[1] ^ buf[2] ^ buf[3]
	_, err := a.writeChar.WriteWithoutResponse(append(buf, xor))
	return err
}

func (a *AkuScale) SetSleepTimeout(ctx context.Context, d time.Duration) error {
	//TODO implement me
	panic("implement me")
}

func (a *AkuScale) ReadBatteryChargePercent(ctx context.Context) (uint8, error) {
	//TODO implement me
	panic("implement me")
}

func (a *AkuScale) setupCharacteristics() error {
	log.Println("Discovering services...")
	services, err := a.btDevice.DiscoverServices([]bluetooth.UUID{comms.AkuServiceUUID})
	if err != nil {
		return fmt.Errorf("could not discover services: %w", err)
	}

	if len(services) == 0 {
		return errors.New("could not find the Lunar BT service")
	}

	for _, service := range services {
		log.Printf("found service %v, scanning for write char", service)
		chars, err := service.DiscoverCharacteristics([]bluetooth.UUID{
			comms.AkuCommandCharUUID,
			comms.AkuNotifyCharUUID,
		})

		if err != nil || len(chars) != 2 {
			return fmt.Errorf("could not discover characteristics: %w", err)
		}

		for _, char := range chars {
			if char.UUID() == comms.AkuCommandCharUUID {
				a.writeChar = char
			}
			if char.UUID() == comms.AkuNotifyCharUUID {
				a.notifyChar = char
			}
		}
	}

	log.Println("Successfully set up characteristics.")
	return nil
}

func (a *AkuScale) handleNotification(buf []byte) {
	a.lastNotified = time.Now()
	weight, ok := comms.DecodeStatusUpdate(buf)
	if !ok {
		log.Printf("unable to decode raw data from notification")
	}
	a.weightUpdateChan <- goscale.WeightUpdate{Value: weight}
}

func (a *AkuScale) setupNotifications() error {
	err := a.notifyChar.EnableNotifications(a.handleNotification)
	if err != nil {
		return fmt.Errorf("failed to enable notifications: %w", err)
	}

	return nil
}
