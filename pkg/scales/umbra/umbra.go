// Package umbra implements a goscale.Scale driver for the Acaia Lunar Umbra.
//
// The Umbra speaks the same Acaia framing protocol as the Lunar but with
// different BLE UUIDs and a big-endian byte order on the raw weight value.
// It also does not require periodic heartbeats to keep the connection alive,
// so this driver omits the heartbeat goroutine that the Lunar driver runs
// and instead relies on the natural notification stream plus a connectivity
// watchdog to detect a dead link.
package umbra

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mlsorensen/goscale"
	"github.com/mlsorensen/goscale/pkg/scales/umbra/comms"
	"tinygo.org/x/bluetooth"
)

func init() {
	goscale.Register("UMBRA", New)
}

var _ goscale.Scale = (*UmbraScale)(nil)

var features = goscale.ScaleFeatures{
	Tare:           true,
	BatteryPercent: true,
	SleepTimeout:   true,
	Beep:           true,
}

type UmbraScale struct {
	name           string
	address        bluetooth.Address
	disconnectCtx  context.Context
	disconnectFunc context.CancelFunc

	btDevice   bluetooth.Device
	writeChar  bluetooth.DeviceCharacteristic
	notifyChar bluetooth.DeviceCharacteristic

	weightUpdateChan chan goscale.WeightUpdate

	lastNotified time.Time
	isConnected  bool

	status comms.StatusMessage
}

func New(device *goscale.FoundDevice) goscale.Scale {
	return &UmbraScale{
		name:    device.Name,
		address: device.Address,
	}
}

func (u *UmbraScale) GetFeatures() goscale.ScaleFeatures {
	return features
}

func (u *UmbraScale) IsConnected() bool {
	return u.isConnected
}

func (u *UmbraScale) DeviceName() string {
	return u.name
}

func (u *UmbraScale) DisplayName() string {
	return "Acaia Lunar Umbra Scale"
}

func (u *UmbraScale) GetSleepTimeout() string {
	return u.status.SleepTimerSetting.String()
}

func (u *UmbraScale) Connect() (<-chan goscale.WeightUpdate, error) {
	if err := goscale.TryEnableAdapter(); err != nil {
		return nil, err
	}

	u.weightUpdateChan = make(chan goscale.WeightUpdate, 20)
	u.disconnectCtx, u.disconnectFunc = context.WithCancel(context.Background())

	var err error
	u.btDevice, err = goscale.BTAdapter.Connect(u.address, bluetooth.ConnectionParams{})
	if err != nil {
		return nil, err
	}

	if err := u.setupCharacteristics(); err != nil {
		_ = u.Disconnect()
		return nil, err
	}

	log.Println("setting up notifications")
	if err := u.setupNotifications(); err != nil {
		_ = u.Disconnect()
		return nil, err
	}

	u.lastNotified = time.Now()
	u.isConnected = true

	// Fast disconnect detection: hook the BLE link's HCI Disconnection
	// Complete event (fires within ~2s of the scale powering off via the
	// link supervision timeout). The handler simply cancels our context;
	// a watchdog goroutine then runs Disconnect off the bluetooth event
	// thread to avoid recursing back into the bluetooth lib.
	goscale.BTAdapter.SetConnectHandler(func(d bluetooth.Device, connected bool) {
		if !connected && u.disconnectFunc != nil {
			u.disconnectFunc()
		}
	})

	// Watchdog: react to either an externally-triggered Disconnect (via
	// disconnectCtx) or a long stretch of silence (fallback in case the
	// HCI disconnect event doesn't fire for some reason).
	go func() {
		const idleLimit = 30 * time.Second
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-u.disconnectCtx.Done():
				_ = u.Disconnect()
				return
			case <-t.C:
				if time.Now().After(u.lastNotified.Add(idleLimit)) {
					log.Println("Umbra: no notifications for", idleLimit, "— disconnecting")
					_ = u.Disconnect()
					return
				}
			}
		}
	}()

	return u.weightUpdateChan, nil
}

func (u *UmbraScale) Disconnect() error {
	// Idempotent — multiple producers (watchdog goroutine, ctx.Done case,
	// external scale.Driver) can race into Disconnect. Closing the update
	// channel twice panics.
	if !u.isConnected {
		return nil
	}
	u.isConnected = false

	err := u.btDevice.Disconnect()
	if u.weightUpdateChan != nil {
		close(u.weightUpdateChan)
		u.weightUpdateChan = nil
	}
	if u.disconnectFunc != nil {
		u.disconnectFunc()
	}
	return err
}

func (u *UmbraScale) Tare(blocking bool) error {
	_, err := u.writeChar.WriteWithoutResponse(comms.TareCommand)
	return err
}

func (u *UmbraScale) AdvanceSleepTimeout() error {
	timeout := comms.AutoOffDisabled
	if u.status.SleepTimerSetting != comms.AutoOffMaxSetting {
		timeout = u.status.SleepTimerSetting + 1
	}

	_, err := u.writeChar.WriteWithoutResponse(comms.BuildAutoOffCommand(timeout))
	if err != nil {
		return fmt.Errorf("error while writing new sleep timeout: %v", err)
	}
	return nil
}

func (u *UmbraScale) SetBeep(beep bool) error {
	_, err := u.writeChar.WriteWithoutResponse(comms.BuildSetBeepCommand(beep))
	if err != nil {
		return fmt.Errorf("error while writing new beep setting: %v", err)
	}
	return nil
}

func (u *UmbraScale) GetBeep() bool {
	return u.status.SoundSetting.Boolean()
}

func (u *UmbraScale) GetBatteryChargePercent() (float64, error) {
	return u.status.Battery, nil
}

func (u *UmbraScale) setupNotifications() error {
	if err := u.notifyChar.EnableNotifications(u.handleNotification); err != nil {
		return fmt.Errorf("failed to enable notifications: %w", err)
	}

	log.Println("initiating handshake")
	// Umbra's command char only supports Write Without Response (ATT Write
	// Command), unlike the Lunar which requires Write Request.
	if _, err := u.writeChar.WriteWithoutResponse(comms.IdentifyCommand); err != nil {
		return fmt.Errorf("failed to send initial handshake: %w", err)
	}

	if _, err := u.writeChar.WriteWithoutResponse(comms.NotificationRequestCommand); err != nil {
		return fmt.Errorf("failed to send notification request: %w", err)
	}

	return nil
}

func (u *UmbraScale) setupCharacteristics() error {
	log.Println("Discovering services...")
	services, err := u.btDevice.DiscoverServices([]bluetooth.UUID{comms.UmbraServiceUUID})
	if err != nil {
		return fmt.Errorf("could not discover services: %w", err)
	}

	if len(services) == 0 {
		return errors.New("could not find the Umbra BT service")
	}

	for _, service := range services {
		log.Printf("found service %v, scanning for write char", service)
		chars, err := service.DiscoverCharacteristics([]bluetooth.UUID{
			comms.UmbraCommandCharUUID,
			comms.UmbraNotifyCharUUID,
		})

		if err != nil || len(chars) != 2 {
			return fmt.Errorf("could not discover characteristics: %w", err)
		}

		for _, char := range chars {
			if char.UUID() == comms.UmbraCommandCharUUID {
				u.writeChar = char
			}
			if char.UUID() == comms.UmbraNotifyCharUUID {
				u.notifyChar = char
			}
		}
	}

	log.Println("Successfully set up characteristics.")
	return nil
}

// handleNotification is the callback for all incoming BLE data.
func (u *UmbraScale) handleNotification(buf []byte) {
	u.lastNotified = time.Now()

	msg, err := comms.DecodeNotification(buf)
	if err != nil {
		log.Printf("[HANDLER] Failed to parse notification: %v. Data: % X", err, buf)
		return
	}

	switch t := msg.(type) {
	case comms.WeightMessage:
		if u.weightUpdateChan != nil {
			u.weightUpdateChan <- goscale.WeightUpdate{Value: t.Weight}
		}
	case comms.StatusMessage:
		u.status = t
		log.Printf("----> Got settings update: %v", t)
	case comms.DeviceInfoMessage:
		log.Printf("---> Got device info: %v", t)
	case comms.UnhandledMessage:
		if t.MsgType != nil {
			log.Printf("--> Unhandled Nested Message. Type: %d. Raw Frame: % X", *t.MsgType, t.RawFrame)
		} else {
			log.Printf("--> Unhandled Command. ID: 0x%X. Raw Frame: % X", t.CommandID, t.RawFrame)
		}
	default:
		log.Printf("--> Unknown decoded message type: %T", msg)
	}
}
