package goscale

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// WeightUpdate represents a single reading from the scale.
// It includes the value, unit, and a flag indicating if the weight is stable.
// An error can be propagated through the channel as well.
type WeightUpdate struct {
	Value float64
	Unit  string
	Error error
}

// Scale is the generic interface for a Bluetooth scale.
// Implementations of this interface will handle communication with a specific model.
type Scale interface {
	// Connect establishes a connection to the scale. Context should be handled internally
	// between the connect and disconnect functions. Returns a read-only
	// channel for weight updates.
	Connect() (<-chan WeightUpdate, error)

	// Disconnect terminates the connection.
	Disconnect() error

	// Tare zeros the scale. If blocking is true, the function will wait for
	// confirmation from the scale before returning, providing confidence the scale is
	// zeroed before proceeding
	Tare(blocking bool) error

	// SetSleepTimeout sets the auto-off timer for the scale.
	SetSleepTimeout(ctx context.Context, d time.Duration) error

	// ReadBatteryChargePercent returns the current battery level as a percentage (0-100).
	ReadBatteryChargePercent(ctx context.Context) (uint8, error)
}

// --- Implementation Registry ---

// Factory is a function that creates a new instance of a Scale.
type Factory func(*FoundDevice) Scale

var (
	registry = make(map[string]Factory)
	regLock  = sync.RWMutex{}
)

// Register makes a scale implementation available by its device name prefix.
// This function should be called from the init() function of the implementation's package.
// For example, an implementation for a "LUNAR" scale would register with the prefix "LUNAR".
func Register(namePrefix string, factory Factory) {
	regLock.Lock()
	defer regLock.Unlock()

	if _, found := registry[namePrefix]; found {
		// Or panic, depending on desired strictness
		fmt.Printf("warning: scale implementation for prefix '%s' is being overwritten\n", namePrefix)
	}
	registry[namePrefix] = factory
}

// NewScaleForDevice finds a registered factory for the given device name and
// creates a new Scale instance. It matches based on the prefix.
// Example: A device named "LUNAR-A23B" would match a registered "LUNAR" prefix.
func NewScaleForDevice(device *FoundDevice) (Scale, error) {
	regLock.RLock()
	defer regLock.RUnlock()

	for prefix, factory := range registry {
		if strings.HasPrefix(device.Name, prefix) {
			return factory(device), nil
		}
	}

	return nil, fmt.Errorf("no implementation found for device '%s'", device.Name)
}
