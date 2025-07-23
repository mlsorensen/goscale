package comms

import "fmt"

// Unit represents the unit of measurement for the scale.
type Unit uint8

const (
	UnitGrams  Unit = 2 // Standard grams
	UnitOunces Unit = 5 // Standard ounces
)

// Constants for the communication protocol.
const (
	// HeaderPrefix1 is the first byte of the message header.
	HeaderPrefix1 byte = 0xEF
	// HeaderPrefix2 is the second byte of the message header.
	HeaderPrefix2 byte = 0xDD
)

type LunarMessage interface{}

type UnhandledMessage struct {
	CommandID byte
	MsgType   *byte // Can be nil if not a nested message
	Payload   []byte
	RawFrame  []byte // Add this field
}

// String returns a formatted version string, e.g., "1.0.18".
func (v FirmwareVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Main, v.Sub, v.Add)
}

// FirmwareVersion holds the main, sub, and additional version numbers for the device firmware.
type FirmwareVersion struct {
	Main uint8
	Sub  uint8
	Add  uint8
}

// DeviceInfoMessage holds the parsed device information from a type 7 info event message.
type DeviceInfoMessage struct {
	Firmware      FirmwareVersion
	IsPasswordSet bool
}

// ScaleMode represents the operational mode of the scale.
type ScaleMode uint8

// Exact modes can vary by scale model
const (
	Mode1Weighing           ScaleMode = 0
	Mode2DualDisplay        ScaleMode = 1
	Mode3PourOver           ScaleMode = 2
	Mode4Espresso           ScaleMode = 3
	Mode5EspressoEarlyTimer ScaleMode = 4
	Mode6AutoTareOnly       ScaleMode = 5
)

// WeightType indicates the type of weight being reported.
type WeightType uint8

const (
	WeightTypeNet   WeightType = 0 // Net weight (this implies the scale is currently tared)
	WeightTypeGross WeightType = 1 // Gross weight (pw, possibly "platform weight")
	WeightTypeTare  WeightType = 2 // ??
)

func (w WeightType) String() string {
	switch w {
	case WeightTypeNet:
		return "Net"
	case WeightTypeGross:
		return "Gross"
	case WeightTypeTare:
		return "Tare"
	default:
		return fmt.Sprintf("Unknown (%d)", w)
	}
}

// WeightMessage holds the complete, parsed weight information from the scale.
type WeightMessage struct {
	Weight   float64
	Type     WeightType
	IsStable bool // True if the weight reading is stable.
}

func (u Unit) String() string {
	switch u {
	case UnitGrams:
		return "grams"
	case UnitOunces:
		return "ounces"
	default:
		return fmt.Sprintf("Unknown Unit (%d)", u)
	}
}

func (m ScaleMode) String() string {
	switch m {
	case Mode1Weighing:
		return "Mode 1: Weighing"
	case Mode2DualDisplay:
		return "Mode 2: Dual Display"
	case Mode3PourOver:
		return "Mode 3: Pour Over"
	case Mode4Espresso:
		return "Mode 4: Espresso"
	case Mode5EspressoEarlyTimer:
		return "Mode 5: Espresso + Immediate Timer"
	case Mode6AutoTareOnly:
		return "Mode 6: Auto-Tare Only"
	default:
		return fmt.Sprintf("Unknown Mode (%d)", m)
	}
}

// AutoOffSetting represents the scale's auto-off timer duration.
type AutoOffSetting uint8

const (
	AutoOffDisabled AutoOffSetting = 0 // Auto-off is disabled
	AutoOff5Min     AutoOffSetting = 1 // Auto-off after 5 minutes
	AutoOff10Min    AutoOffSetting = 2 // Auto-off after 10 minutes
	AutoOff20Min    AutoOffSetting = 3 // Auto-off after 20 minutes
	AutoOff30Min    AutoOffSetting = 4 // Auto-off after 30 minutes
	AutoOff60Min    AutoOffSetting = 5 // Auto-off after 60 minutes
)

func (s AutoOffSetting) String() string {
	switch s {
	case AutoOffDisabled:
		return "Disabled"
	case AutoOff5Min:
		return "5 Minutes"
	case AutoOff10Min:
		return "10 Minutes"
	case AutoOff20Min:
		return "20 Minutes"
	case AutoOff30Min:
		return "30 Minutes"
	case AutoOff60Min:
		return "60 Minutes"
	default:
		return fmt.Sprintf("Unknown Setting (%d)", s)
	}
}

// SoundSetting represents the scale's beep sound setting.
type SoundSetting uint8

const (
	SoundOff SoundSetting = 0 // Beep sound is off
	SoundOn  SoundSetting = 1 // Beep sound is on
)

func (s SoundSetting) String() string {
	if s == SoundOn {
		return "On"
	}
	return "Off"
}

// KeyDisableSetting represents the scale's key lock timer. Not really sure how to use this.
type KeyDisableSetting uint8

const (
	KeyDisableOff KeyDisableSetting = 0 // Keys are never disabled
	KeyDisable10s KeyDisableSetting = 1 // Keys disabled after 10 seconds
	KeyDisable20s KeyDisableSetting = 2 // Keys disabled after 20 seconds
	KeyDisable30s KeyDisableSetting = 3 // Keys disabled after 30 seconds
)

func (k KeyDisableSetting) String() string {
	switch k {
	case KeyDisableOff:
		return "Off"
	case KeyDisable10s:
		return "10 Seconds"
	case KeyDisable20s:
		return "20 Seconds"
	case KeyDisable30s:
		return "30 Seconds"
	default:
		return fmt.Sprintf("Unknown Setting (%d)", k)
	}
}

// ResolutionSetting represents the scale's measurement precision.
type ResolutionSetting uint8

const (
	ResolutionLow  ResolutionSetting = 0 // Low precision (e.g., .1g increments)
	ResolutionHigh ResolutionSetting = 1 // High precision (e.g., 0.01g increments)
)

func (r ResolutionSetting) String() string {
	if r == ResolutionHigh {
		return "High"
	}
	return "Low"
}

// CapacitySetting represents the scale's maximum weight capacity.
type CapacitySetting uint8

const (
	Capacity1000g CapacitySetting = 0 // 1000g max capacity
	Capacity2000g CapacitySetting = 1 // 2000g max capacity
)

func (c CapacitySetting) String() string {
	if c == Capacity2000g {
		return "2000g"
	}
	return "1000g"
}

// StatusMessage holds the parsed settings from a type 8 status event message from an Acaia scale.
type StatusMessage struct {
	StatusLength       uint8
	Battery            float64           // Battery level percentage (0.0-100.0)
	IsTimerRunning     bool              // True if the timer is currently running
	Unit               Unit              // The unit of measurement
	IsCountdownRunning bool              // True if the countdown is active
	ScaleMode          ScaleMode         // The current mode of the scale
	IsTared            bool              // True if the scale is tared
	SleepTimerSetting  AutoOffSetting    // Auto-sleep timer setting
	KeyDisableSetting  KeyDisableSetting // Key disable setting
	SoundSetting       SoundSetting      // Beep sound setting
	ResolutionSetting  ResolutionSetting // Display resolution setting
	CapacitySetting    CapacitySetting   // Scale capacity setting
	TimerValue         uint16            // Timer value in seconds, if present
}
