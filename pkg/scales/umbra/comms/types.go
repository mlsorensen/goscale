package comms

import "fmt"

// Unit represents the unit of measurement for the scale.
//
// Umbra uses a different unit-byte mapping than Lunar (Lunar: g=2, oz=5;
// Umbra reports g=0). The ounce mapping below is an educated guess — toggle
// the unit on the scale and watch the status to confirm.
type Unit uint8

const (
	UnitGrams  Unit = 0
	UnitOunces Unit = 1
)

const (
	HeaderPrefix1 byte = 0xEF
	HeaderPrefix2 byte = 0xDD
)

type UmbraMessage interface{}

type UnhandledMessage struct {
	CommandID byte
	MsgType   *byte
	Payload   []byte
	RawFrame  []byte
}

func (v FirmwareVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Main, v.Sub, v.Add)
}

type FirmwareVersion struct {
	Main uint8
	Sub  uint8
	Add  uint8
}

type DeviceInfoMessage struct {
	Firmware      FirmwareVersion
	IsPasswordSet bool
}

type ScaleMode uint8

const (
	Mode1Weighing           ScaleMode = 0
	Mode2DualDisplay        ScaleMode = 1
	Mode3PourOver           ScaleMode = 2
	Mode4Espresso           ScaleMode = 3
	Mode5EspressoEarlyTimer ScaleMode = 4
	Mode6AutoTareOnly       ScaleMode = 5
)

type WeightType uint8

const (
	WeightTypeNet   WeightType = 0
	WeightTypeGross WeightType = 1
	WeightTypeTare  WeightType = 2
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

type WeightMessage struct {
	Weight   float64
	Type     WeightType
	IsStable bool
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

// AutoOffSetting represents the Umbra's combined sleep / auto-off timer
// setting. The Umbra distinguishes "sleep" (display off, scale stays on)
// from "auto-off" (powers off entirely) and packs both into one enum.
// Values come from AUTO_OFF_SETTING_UMBRA in the Acaia Android SDK.
type AutoOffSetting uint8

const (
	AutoOffDisabled  AutoOffSetting = 0 // No timer
	AutoOffSleep5M   AutoOffSetting = 1 // Sleep after 5 minutes
	AutoOffSleep10M  AutoOffSetting = 2 // Sleep after 10 minutes
	AutoOffSleep30M  AutoOffSetting = 3 // Sleep after 30 minutes
	AutoOffPower5M   AutoOffSetting = 4 // Power off after 5 minutes
	AutoOffPower10M  AutoOffSetting = 5 // Power off after 10 minutes
	AutoOffPower30M  AutoOffSetting = 6 // Power off after 30 minutes
	AutoOffSleep1M   AutoOffSetting = 7 // Sleep after 1 minute
	AutoOffMaxSetting               = AutoOffSleep1M
)

func (s AutoOffSetting) String() string {
	switch s {
	case AutoOffDisabled:
		return "Disabled"
	case AutoOffSleep1M:
		return "Sleep after 1m"
	case AutoOffSleep5M:
		return "Sleep after 5m"
	case AutoOffSleep10M:
		return "Sleep after 10m"
	case AutoOffSleep30M:
		return "Sleep after 30m"
	case AutoOffPower5M:
		return "Power off after 5m"
	case AutoOffPower10M:
		return "Power off after 10m"
	case AutoOffPower30M:
		return "Power off after 30m"
	default:
		return fmt.Sprintf("Unknown Setting (%d)", s)
	}
}

type SoundSetting uint8

const (
	SoundOff SoundSetting = 0
	SoundOn  SoundSetting = 1
)

func (s SoundSetting) String() string {
	if s == SoundOn {
		return "On"
	}
	return "Off"
}

func (s SoundSetting) Boolean() bool {
	return s == SoundOn
}

type KeyDisableSetting uint8

const (
	KeyDisableOff KeyDisableSetting = 0
	KeyDisable10s KeyDisableSetting = 1
	KeyDisable20s KeyDisableSetting = 2
	KeyDisable30s KeyDisableSetting = 3
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

type ResolutionSetting uint8

const (
	ResolutionLow  ResolutionSetting = 0
	ResolutionHigh ResolutionSetting = 1
)

func (r ResolutionSetting) String() string {
	if r == ResolutionHigh {
		return "High"
	}
	return "Low"
}

type CapacitySetting uint8

const (
	Capacity1000g CapacitySetting = 0
	Capacity2000g CapacitySetting = 1
)

func (c CapacitySetting) String() string {
	if c == Capacity2000g {
		return "2000g"
	}
	return "1000g"
}

// StatusMessage holds the Umbra's settings status. The Umbra reports a
// distinct 13-byte payload from the Lunar (separate per-field bytes, no
// packed flag bits), with extra fields for magic-relay and firmware version.
// See ScaleProtocol.status_event_umbra in the Acaia Android SDK.
type StatusMessage struct {
	StatusLength      uint8
	Battery           float64
	SleepTimerSetting AutoOffSetting
	SoundSetting      SoundSetting
	Unit              Unit
	ResolutionSetting ResolutionSetting
	MagicRelaySense   uint8
	MagicRelayBeep    uint8
	Firmware          FirmwareVersion
}
