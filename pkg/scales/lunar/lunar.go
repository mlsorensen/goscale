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
// *MockScale ever stops satisfying the goscale.Scale interface.
var _ goscale.Scale = (*LunarScale)(nil)

type LunarScale struct {
	name           string
	address        bluetooth.Address
	disconnectCtx  context.Context
	disconnectFunc context.CancelFunc

	btDevice   bluetooth.Device
	writeChar  bluetooth.DeviceCharacteristic
	notifyChar bluetooth.DeviceCharacteristic

	weightUpdateChan chan goscale.WeightUpdate
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

	l.btDevice, err = goscale.BTAdapter.Connect(l.address, bluetooth.ConnectionParams{
		MaxInterval: bluetooth.Duration(1000),
		MinInterval: bluetooth.Duration(10),
	})

	if err != nil {
		return nil, err
	}

	err = l.setupCharacteristics()
	if err != nil {
		_ = l.Disconnect()
		return nil, err
	}

	log.Println("setting up notifications")

	err = l.notifyChar.EnableNotifications(l.handleNotification)
	if err != nil {
		_ = l.Disconnect()
		return nil, fmt.Errorf("failed to enable notifications: %w", err)
	}

	log.Println("initiating handshake")
	_, err = l.writeChar.WriteWithoutResponse(comms.BuildIdentifyCommand())
	if err != nil {
		_ = l.Disconnect()
		return nil, fmt.Errorf("failed to send initial handshake: %w", err)
	}

	// Start the heartbeat goroutine
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		var consecutiveErrors int = 0

		for {
			select {
			case <-ticker.C:
				// Send heartbeat signal to the scale
				if err := l.sendHeartbeat(); err != nil {
					consecutiveErrors++
					log.Printf("Error sending heartbeat: %v", err)
					if consecutiveErrors >= 1 {
						log.Printf("Reached limit to heartbeat errors, disconnecting")
						_ = l.Disconnect()
					}
				}
			case <-l.disconnectCtx.Done():
				_ = l.Disconnect()
				return
			}
		}
	}()

	return l.weightUpdateChan, nil
}

func (l *LunarScale) Disconnect() error {
	l.disconnectFunc()
	return l.btDevice.Disconnect()
}

func (l *LunarScale) Tare(blocking bool) error {
	_, err := l.writeChar.WriteWithoutResponse(comms.BuildTareCommand())
	return err
}

func (l *LunarScale) SetSleepTimeout(ctx context.Context, d time.Duration) error {
	//TODO implement me
	panic("implement me")
}

func (l *LunarScale) ReadBatteryChargePercent(ctx context.Context) (uint8, error) {
	//TODO implement me
	panic("implement me")
}

func (l *LunarScale) sendHeartbeat() error {
	log.Printf("sending heartbeat")
	_, err := l.writeChar.WriteWithoutResponse(comms.BuildGetStatusCommand())

	_, err = l.writeChar.WriteWithoutResponse(comms.BuildNotificationRequestCommand())
	if err != nil {
		_ = l.Disconnect()
		return fmt.Errorf("failed to send notification request: %w", err)
	}

	if err != nil {
		return err
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
	packet, err := comms.ParseNotification(buf)
	if err != nil {
		log.Printf("[HANDLER] Failed to parse notification: %v. Data: % X", err, buf)
		return
	}

	// If we get here, 'packet' is a valid, decoded message.
	log.Printf("[HANDLER] Decoded packet: %#v", packet)

	// Use a type switch to handle the specific, decoded packet type.
	switch p := packet.(type) {
	case comms.WeightPacket:
		log.Printf("--> Weight Update: %.2f", p.Weight)
		// Send the update to the user's channel.
		l.weightUpdateChan <- goscale.WeightUpdate{Value: p.Weight}
	case comms.UnhandledPacket:
		// This is the updated logging case
		if p.MsgType != nil {
			// It was an unhandled nested message (from command 12)
			log.Printf("--> Unhandled Nested Message. Type: %d. Raw Frame: % X", *p.MsgType, p.RawFrame)
		} else {
			// It was an unhandled top-level command
			log.Printf("--> Unhandled Command. ID: 0x%X. Raw Frame: % X", p.CommandID, p.RawFrame)
		}
	default:
		// This default case is a fallback for unexpected parsed types
		log.Printf("--> Unknown packet type after successful parsing. Raw Data: % X", buf)
	}
	time.Sleep(50 * time.Millisecond)
}
