// Package mock provides a mock implementation of the goscale.Scale interface.
// It is intended for development and testing purposes when a physical scale is not available.
package mock

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
	"tinygo.org/x/bluetooth"

	"github.com/mlsorensen/goscale"
)

// This init function registers the MockScale with the central registry.
// To use it, you must explicitly import this package.
func init() {
	// Register with a distinct name, "MOCK", so it can be requested specifically.
	goscale.Register("MOCK", New)
}

// This line is the compile-time check. It will fail to compile if
// *MockScale ever stops satisfying the goscale.Scale interface.
var _ goscale.Scale = (*MockScale)(nil)
var features = goscale.ScaleFeatures{
	Tare:           true,
	BatteryPercent: true,
	SleepTimeout:   true,
}

// MockScale is a simulated Bluetooth scale for development.
type MockScale struct {
	name         string
	address      bluetooth.Address
	mu           sync.Mutex
	connected    bool
	batteryLevel float64
	weight       float64

	disconnectCtx context.Context
	disconnect    context.CancelFunc

	// Channels to control the simulation goroutine
	stopChan      chan struct{}
	tareRequested chan struct{}
}

func (s *MockScale) GetFeatures() goscale.ScaleFeatures {
	return features
}

func (s *MockScale) IsConnected() bool {
	return s.connected
}

func (s *MockScale) DeviceName() string {
	return s.name
}

func (s *MockScale) DisplayName() string {
	return "Mock Scale"
}

// New creates a new, uninitialized MockScale.
func New(device *goscale.FoundDevice) goscale.Scale {
	return &MockScale{
		name:         device.Name,
		address:      bluetooth.Address{},
		batteryLevel: .98,  // Start with a high battery
		weight:       21.5, // Start with some initial weight
	}
}

// Connect starts the simulation.
func (s *MockScale) Connect() (<-chan goscale.WeightUpdate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.disconnectCtx, s.disconnect = context.WithCancel(context.Background())

	if s.connected {
		return nil, fmt.Errorf("mock scale is already connected")
	}

	log.Println("MOCK: Connecting...")
	s.connected = true
	s.stopChan = make(chan struct{})
	s.tareRequested = make(chan struct{})

	updates := make(chan goscale.WeightUpdate)

	// Start the simulation goroutine
	go s.simulate(s.disconnectCtx, updates)

	log.Println("MOCK: Connected successfully.")
	return updates, nil
}

// simulate is the core loop that generates fake data.
func (s *MockScale) simulate(ctx context.Context, updates chan<- goscale.WeightUpdate) {
	// IMPORTANT: Ensure the channel is closed on exit to signal disconnection.
	defer close(updates)
	defer log.Println("MOCK: Simulation stopped.")

	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			// Add a small random drift to the weight
			s.weight += (rand.Float64() - 0.4) * 0.5 // a little up, a little down
			if s.weight < 0 {
				s.weight = 0
			}
			update := goscale.WeightUpdate{
				Value: s.weight,
				Unit:  "g",
			}
			s.mu.Unlock()
			updates <- update

		case <-s.tareRequested:
			log.Println("MOCK: Tare requested, resetting weight to 0.")
			s.mu.Lock()
			s.weight = 0
			s.mu.Unlock()
			// Send an immediate update after taring
			updates <- goscale.WeightUpdate{Value: 0, Unit: "g"}

		case <-s.stopChan: // Disconnect() was called
			return

		case <-ctx.Done(): // Parent context was cancelled
			return
		}
	}
}

// Disconnect stops the simulation.
func (s *MockScale) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil // Nothing to do
	}

	s.disconnect()

	log.Println("MOCK: Disconnecting...")
	if s.stopChan != nil {
		close(s.stopChan)
		s.stopChan = nil
	}
	s.connected = false
	log.Println("MOCK: Disconnected.")
	return nil
}

// Tare sends a request to the simulation to zero the weight.
func (s *MockScale) Tare(blocking bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return fmt.Errorf("mock scale is not connected")
	}

	// Send the tare request without blocking the mutex
	go func() {
		s.tareRequested <- struct{}{}
	}()

	if blocking {
		// In a mock, we can just sleep to simulate the round trip time.
		log.Println("MOCK: Tare is blocking, waiting for simulation...")
		time.Sleep(250 * time.Millisecond)
	}
	return nil
}

// SetSleepTimeout just logs the action.
func (s *MockScale) AdvanceSleepTimeout() error {
	log.Printf("MOCK: SetSleepTimeout called")
	return nil
}

// ReadBatteryChargePercent returns the simulated battery level.
func (s *MockScale) GetBatteryChargePercent() (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Println("MOCK: Reading battery level.")
	return s.batteryLevel, nil
}

func (s *MockScale) GetSleepTimeout() string {
	return "never"
}

func (s *MockScale) SetBeep(b bool) error {
	log.Println("BEEP")
	return nil
}

func (s *MockScale) GetBeep() bool {
	//TODO implement me
	panic("implement me")
}
