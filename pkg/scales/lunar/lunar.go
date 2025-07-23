package lunar

import (
	"context"
	"errors"
	"fmt"
	"github.com/mlsorensen/goscale"
	"github.com/mlsorensen/goscale/pkg/scales/lunar/comms"
	"log"
	"time"
	"tinygo.org/x/bluetooth"
)

func init() {
	// Register with a distinct name, "MOCK", so it can be requested specifically.
	goscale.Register("LUNAR", New)
}

// This line is the compile-time check. It will fail to compile if
// *LunarScale ever stops satisfying the goscale.Scale interface.
var _ goscale.Scale = (*LunarScale)(nil)

var features = goscale.ScaleFeatures{
	Tare:           true,
	BatteryPercent: true,
	SleepTimeout:   true,
	Beep:           true,
}

type LunarScale struct {
	name           string
	address        bluetooth.Address
	disconnectCtx  context.Context
	disconnectFunc context.CancelFunc
	synced         bool

	btDevice   bluetooth.Device
	writeChar  bluetooth.DeviceCharacteristic
	notifyChar bluetooth.DeviceCharacteristic

	weightUpdateChan chan goscale.WeightUpdate

	lastNotified time.Time
	isConnected  bool

	status comms.StatusMessage
}

func (l *LunarScale) GetFeatures() goscale.ScaleFeatures {
	return features
}

func (l *LunarScale) IsConnected() bool {
	return l.isConnected
}

func (l *LunarScale) DeviceName() string {
	return l.name
}

func (l *LunarScale) DisplayName() string {
	return "Acaia Lunar Scale"
}

func (l *LunarScale) GetSleepTimeout() string {
	return l.status.SleepTimerSetting.String()
}

func New(device *goscale.FoundDevice) goscale.Scale {
	return &LunarScale{
		name:    device.Name,
		address: device.Address,
	}
}

// Connect will connect the scale, setting up heartbeat to maintain connection, and return a channel
// for receiving weight updates
func (l *LunarScale) Connect() (<-chan goscale.WeightUpdate, error) {
	err := goscale.TryEnableAdapter()
	if err != nil {
		return nil, err
	}

	l.weightUpdateChan = make(chan goscale.WeightUpdate, 20)

	l.disconnectCtx, l.disconnectFunc = context.WithCancel(context.Background())

	l.btDevice, err = goscale.BTAdapter.Connect(l.address, bluetooth.ConnectionParams{})

	if err != nil {
		return nil, err
	}

	err = l.setupCharacteristics()
	if err != nil {
		_ = l.Disconnect()
		return nil, err
	}

	log.Println("setting up notifications")
	err = l.setupNotifications()
	if err != nil {
		_ = l.Disconnect()
		return nil, err
	}

	l.isConnected = true

	// Start the heartbeat goroutine
	go func() {
		for {
			select {
			case <-l.disconnectCtx.Done():
				_ = l.Disconnect()
				return
			default:
				// Send heartbeat signal to the scale
				if err := l.sendHeartbeat(); err != nil {
					log.Printf("Error sending heartbeat: %v", err)
				}
			}
		}
	}()

	return l.weightUpdateChan, nil
}

func (l *LunarScale) Disconnect() error {
	err := l.btDevice.Disconnect()
	if err != nil {
		// are we still connected or not? who knows
		return err
	}
	//TODO: mutex
	if l.weightUpdateChan != nil {
		close(l.weightUpdateChan)
	}
	l.disconnectFunc()
	l.isConnected = false
	return nil
}

func (l *LunarScale) Tare(blocking bool) error {
	_, err := l.writeChar.WriteWithoutResponse(comms.TareCommand)
	return err
}

func (l *LunarScale) AdvanceSleepTimeout() error {
	timeout := comms.AutoOffDisabled
	if l.status.SleepTimerSetting != 5 {
		timeout = l.status.SleepTimerSetting + 1
	}

	_, err := l.writeChar.WriteWithoutResponse(comms.BuildAutoOffCommand(timeout))
	if err != nil {
		return fmt.Errorf("error while writing new sleep timeout: %v", err)
	}
	return nil
}

func (l *LunarScale) SetBeep(beep bool) error {
	_, err := l.writeChar.WriteWithoutResponse(comms.BuildSetBeepCommand(beep))
	if err != nil {
		return fmt.Errorf("error while writing new beep setting: %v", err)
	}
	return nil
}

func (l *LunarScale) GetBeep() bool {
	return l.status.SoundSetting.Boolean()
}

func (l *LunarScale) GetBatteryChargePercent() (float64, error) {
	return l.status.Battery, nil
}

func (l *LunarScale) sendHeartbeat() error {
	log.Printf("sending heartbeat")
	if !l.isConnected {
		return fmt.Errorf("no heartbeat allowed if not connected")
	}

	if !l.synced {
		_, err := l.writeChar.Write(comms.GetStatusCommand)
		if err != nil {
			log.Printf("Error on heartbeat: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	} else {
		_, err := l.writeChar.Write(comms.GetStatusCommand)
		if err != nil {
			log.Printf("Error on heartbeat: %v", err)
			l.Disconnect()
		}
		time.Sleep(time.Second)
	}

	if l.lastNotified.IsZero() || time.Now().After(l.lastNotified.Add(time.Second)) {
		log.Println("setting up notifications again")
		_ = l.setupNotifications()
	}
	return nil
}

func (l *LunarScale) setupNotifications() error {
	err := l.notifyChar.EnableNotifications(l.handleNotification)
	if err != nil {
		return fmt.Errorf("failed to enable notifications: %w", err)
	}

	log.Println("initiating handshake")
	_, err = l.writeChar.WriteWithoutResponse(comms.IdentifyCommand)
	if err != nil {
		return fmt.Errorf("failed to send initial handshake: %w", err)
	}

	_, err = l.writeChar.WriteWithoutResponse(comms.NotificationRequestCommand)
	if err != nil {
		return fmt.Errorf("failed to send notification request: %w", err)
	}

	return nil
}

func (l *LunarScale) setupCharacteristics() error {
	log.Println("Discovering services...")
	services, err := l.btDevice.DiscoverServices([]bluetooth.UUID{comms.LunarServiceUUID})
	if err != nil {
		return fmt.Errorf("could not discover services: %w", err)
	}

	if len(services) == 0 {
		return errors.New("could not find the Lunar BT service")
	}

	for _, service := range services {
		log.Printf("found service %v, scanning for write char", service)
		chars, err := service.DiscoverCharacteristics([]bluetooth.UUID{
			comms.LunarCommandCharUUID,
			comms.LunarNotifyCharUUID,
		})

		if err != nil || len(chars) != 2 {
			return fmt.Errorf("could not discover characteristics: %w", err)
		}

		for _, char := range chars {
			if char.UUID() == comms.LunarCommandCharUUID {
				l.writeChar = char
			}
			if char.UUID() == comms.LunarNotifyCharUUID {
				l.notifyChar = char
			}
		}
	}

	log.Println("Successfully set up characteristics.")
	return nil
}

// handleNotification is the callback for all incoming BLE data.
// It assumes one notification callback contains one complete message.
func (l *LunarScale) handleNotification(buf []byte) {
	// Attempt to parse the entire buffer as a single message.
	msg, err := comms.DecodeNotification(buf)
	if err != nil {
		log.Printf("[HANDLER] Failed to parse notification: %v. Data: % X", err, buf)
		return
	}

	// If we get here, 'packet' is a valid, decoded message.
	//log.Printf("[HANDLER] Decoded packet: %#v", msg)

	// Use a type switch to handle the specific, decoded packet type.
	switch t := msg.(type) {
	case comms.WeightMessage:
		//log.Printf("--> Weight Update: %v", t)
		// Send the update to the user's channel.
		l.weightUpdateChan <- goscale.WeightUpdate{Value: t.Weight}
		l.lastNotified = time.Now()
	case comms.StatusMessage:
		l.synced = true
		l.status = t
		log.Printf("----> Got settings update: %v", t)
	case comms.DeviceInfoMessage:
		log.Printf("---> Got device info: %v", t)
	case comms.UnhandledMessage:
		// This is the updated logging case
		if t.MsgType != nil {
			// It was an unhandled nested message (from command 12)
			log.Printf("--> Unhandled Nested Message. Type: %d. Raw Frame: % X", *t.MsgType, t.RawFrame)
		} else {
			// It was an unhandled top-level command
			log.Printf("--> Unhandled Command. ID: 0x%X. Raw Frame: % X", t.CommandID, t.RawFrame)
		}
	default:
		// This default case is a fallback for unexpected parsed types
		log.Printf("--> Unknown packet type after successful parsing. Raw Data: % X", buf)
	}
	time.Sleep(50 * time.Millisecond)
}
