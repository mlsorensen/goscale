package themis

import (
	"context"
	"errors"
	"fmt"
	"github.com/mlsorensen/goscale"
	"github.com/mlsorensen/goscale/pkg/scales/themis/comms"
	"log"
	"time"
	"tinygo.org/x/bluetooth"
)

func init() {
	goscale.Register("BOOKOO", New)
}

type ThemisScale struct {
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

	status *comms.StatusUpdate
}

// This line is the compile-time check. It will fail to compile if
// *ThemisScale ever stops satisfying the goscale.Scale interface.
var _ goscale.Scale = (*ThemisScale)(nil)

var features = goscale.ScaleFeatures{
	Tare:           true,
	SleepTimeout:   true,
	Beep:           true,
	BatteryPercent: true,
}

func New(device *goscale.FoundDevice) goscale.Scale {
	return &ThemisScale{
		name:    device.Name,
		address: device.Address,
	}
}

func (t *ThemisScale) GetFeatures() goscale.ScaleFeatures {
	return features
}

func (t *ThemisScale) Connect() (<-chan goscale.WeightUpdate, error) {
	err := goscale.TryEnableAdapter()
	if err != nil {
		return nil, err
	}

	t.weightUpdateChan = make(chan goscale.WeightUpdate, 20)

	t.disconnectCtx, t.disconnectFunc = context.WithCancel(context.Background())

	t.btDevice, err = goscale.BTAdapter.Connect(t.address, bluetooth.ConnectionParams{})

	if err != nil {
		return nil, err
	}

	err = t.setupCharacteristics()
	if err != nil {
		_ = t.Disconnect()
		return nil, err
	}

	log.Println("setting up notifications")
	err = t.setupNotifications()
	if err != nil {
		_ = t.Disconnect()
		return nil, err
	}
	t.lastNotified = time.Now()

	t.connected = true

	// Fast disconnect detection via the BLE link's HCI Disconnection
	// Complete event. The handler cancels our context; the watchdog
	// goroutine below picks it up and runs Disconnect off the bluetooth
	// event thread.
	goscale.BTAdapter.SetConnectHandler(func(d bluetooth.Device, connected bool) {
		if !connected && t.disconnectFunc != nil {
			t.disconnectFunc()
		}
	})

	// Watchdog: react to context cancel (external Disconnect or HCI
	// disconnect event) or to a longer no-notifications fallback.
	go func() {
		const idleLimit = 30 * time.Second
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-t.disconnectCtx.Done():
				_ = t.Disconnect()
				return
			case <-ticker.C:
				if time.Now().After(t.lastNotified.Add(idleLimit)) {
					_ = t.Disconnect()
					return
				}
			}
		}
	}()

	return t.weightUpdateChan, nil
}

func (t *ThemisScale) Disconnect() error {
	// Idempotent: the connectivity-monitor goroutine can race itself
	// (timeout check → Disconnect → ctx.Done case → Disconnect) and also
	// races the external scale.Driver disconnect path. Closing
	// weightUpdateChan twice panics.
	if !t.connected {
		return nil
	}
	t.connected = false

	err := t.btDevice.Disconnect()
	if err != nil {
		// Even if the BLE disconnect failed, treat the channel/context
		// teardown as authoritative — we won't be sending on the channel
		// any more from this side.
	}
	if t.weightUpdateChan != nil {
		close(t.weightUpdateChan)
		t.weightUpdateChan = nil
	}
	if t.disconnectFunc != nil {
		t.disconnectFunc()
	}
	return err
}

func (t *ThemisScale) IsConnected() bool {
	return t.connected
}

func (t *ThemisScale) DeviceName() string {
	return t.name
}

func (t *ThemisScale) DisplayName() string {
	return "BOOKOO Themis scale"
}

func (t *ThemisScale) Tare(blocking bool) error {
	_, err := t.writeChar.Write(comms.ThemisTareCommand)
	return err
}

func (t *ThemisScale) AdvanceSleepTimeout() error {
	timeout := comms.AutoOffSettings.NextWithInt(t.status.StandbyTime)
	cmd := comms.BuildAutoOffCommand(timeout)
	fmt.Printf("sleep timer cmd: % x\n", cmd)
	_, err := t.writeChar.Write(cmd)
	if err != nil {
		return fmt.Errorf("error while writing new sleep timeout: %v", err)
	}
	return nil
}

func (t *ThemisScale) GetSleepTimeout() string {
	return fmt.Sprintf("%d Minutes", t.status.StandbyTime)
}

func (t *ThemisScale) GetBatteryChargePercent() (float64, error) {
	return float64(t.status.PowerPercentage), nil
}

func (t *ThemisScale) SetBeep(b bool) error {
	cmd := comms.BuildChangeBeepCommand(b)
	fmt.Printf("beep cmd: % x\n", cmd)
	_, err := t.writeChar.Write(cmd)
	if err != nil {
		return fmt.Errorf("error while writing new beep setting: %v", err)
	}

	return nil
}

func (t *ThemisScale) GetBeep() bool {
	return t.status.BuzzerGear > 0
}

func (t *ThemisScale) setupCharacteristics() error {
	log.Println("Discovering services...")
	services, err := t.btDevice.DiscoverServices([]bluetooth.UUID{comms.ThemisServiceUUID})
	if err != nil {
		return fmt.Errorf("could not discover services: %w", err)
	}

	if len(services) == 0 {
		return errors.New("could not find the Lunar BT service")
	}

	for _, service := range services {
		log.Printf("found service %v, scanning for write char", service)
		chars, err := service.DiscoverCharacteristics([]bluetooth.UUID{
			comms.ThemisCommandCharUUID,
			comms.ThemisNotifyCharUUID,
		})

		if err != nil || len(chars) != 2 {
			return fmt.Errorf("could not discover characteristics: %w", err)
		}

		for _, char := range chars {
			if char.UUID() == comms.ThemisCommandCharUUID {
				t.writeChar = char
			}
			if char.UUID() == comms.ThemisNotifyCharUUID {
				t.notifyChar = char
			}
		}
	}

	log.Println("Successfully set up characteristics.")
	return nil
}

func (t *ThemisScale) handleNotification(buf []byte) {
	t.lastNotified = time.Now()
	status, ok := comms.DecodeStatusUpdate(buf)
	t.status = status
	if !ok {
		log.Printf("unable to decode raw data from notification")
	}
	t.weightUpdateChan <- goscale.WeightUpdate{Value: status.GramsWeight}
}

func (t *ThemisScale) setupNotifications() error {
	err := t.notifyChar.EnableNotifications(t.handleNotification)
	if err != nil {
		return fmt.Errorf("failed to enable notifications: %w", err)
	}

	return nil
}
