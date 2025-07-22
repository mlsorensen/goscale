package decode

import (
	"fmt"
)

// Unit represents the unit of measurement for the scale.
type Unit uint8

const (
	UnitGrams  Unit = 2 // Standard grams
	UnitOunces Unit = 5 // Standard ounces
)

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

// ScaleMode represents the operational mode of the scale.
type ScaleMode uint8

// Exact modes can vary by scale model
// These are common modes.
const (
	Mode1Weighing           ScaleMode = 0
	Mode2DualDisplay        ScaleMode = 1
	Mode3PourOver           ScaleMode = 2
	Mode4Espresso           ScaleMode = 3
	Mode5EspressoEarlyTimer ScaleMode = 4
	Mode6AutoTareOnly       ScaleMode = 5
)

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

// KeyDisableSetting represents the scale's key lock timer.
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
	ResolutionLow  ResolutionSetting = 0 // Low precision (e.g., 1g increments)
	ResolutionHigh ResolutionSetting = 1 // High precision (e.g., 0.1g increments)
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
// The fields are derived from the 'scale_status' struct in the Acaia SDK.
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

// DecodeStatusMessage parses the 9-byte or 12-byte payload from a type 8 event message
// and returns a StatusMessage struct.
func DecodeStatusMessage(payload []byte) (StatusMessage, error) {
	if len(payload) != 9 && len(payload) != 12 {
		return StatusMessage{}, fmt.Errorf("invalid payload length: expected 9 or 12, got %d", len(payload))
	}

	msg := StatusMessage{}

	// Byte 0: Status Length
	// The length of the status payload itself.
	msg.StatusLength = payload[0]

	// Byte 1: Battery and Timer Status
	// This byte contains two fields packed using bitwise operations.
	// n_battery (7 bits): The battery level (0-100).
	// b_timer_start (1 bit): Indicates if the timer is running.
	msg.Battery = float64(payload[1] & 0x7F)       // Lower 7 bits for battery
	msg.IsTimerRunning = (payload[1]>>7)&0x01 == 1 // Most significant bit for timer status

	// Byte 2: Unit and Countdown Status
	// n_unit (7 bits): The unit of measurement.
	// b_cd_start (1 bit): Indicates if the countdown is active.
	msg.Unit = Unit(payload[2] & 0x7F)                 // Lower 7 bits for unit
	msg.IsCountdownRunning = (payload[2]>>7)&0x01 == 1 // Most significant bit for countdown status

	// Byte 3: Scale Mode and Tare Status
	// n_scale_mode (7 bits): The current mode of the scale.
	// b_tare (1 bit): Indicates if the scale is tared.
	msg.ScaleMode = ScaleMode(payload[3] & 0x7F) // Lower 7 bits for scale mode
	msg.IsTared = (payload[3]>>7)&0x01 == 1      // Most significant bit for tare status

	// Byte 4: Sleep Timer Setting
	msg.SleepTimerSetting = AutoOffSetting(payload[4])

	// Byte 5: Key Disable Setting
	msg.KeyDisableSetting = KeyDisableSetting(payload[5])

	// Byte 6: Sound Setting
	msg.SoundSetting = SoundSetting(payload[6])

	// Byte 7: Resolution Setting
	msg.ResolutionSetting = ResolutionSetting(payload[7])

	// Byte 8: Capacity Setting
	msg.CapacitySetting = CapacitySetting(payload[8])

	// Check for optional timer data if the payload is 12 bytes long
	if len(payload) == 12 {
		// The timer value is a 16-bit little-endian integer.
		// It's constructed from payload[9] (minutes), payload[10] (seconds), and payload[11] (deciseconds).
		// For simplicity, we can combine them into a single value. A more complex struct could also be used.
		// Based on the Java SDK, the timer value is often packed.
		// Let's assume a simple seconds representation for now from the extra bytes.
		// For example, if it represents time in some way.
		// A common pattern is minutes and seconds. Let's assume payload[9] is minutes and payload[10] is seconds
		// The structure tm_event in Java suggests minutes, seconds, and deciseconds.
		minutes := uint16(payload[9])
		seconds := uint16(payload[10])
		// deciseconds := uint16(payload[11])

		msg.TimerValue = (minutes * 60) + seconds
	}

	return msg, nil
}
