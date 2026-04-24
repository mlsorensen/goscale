package comms

import "fmt"

// Unit represents the unit of measurement for the scale.
type Unit uint8

const (
	UnitGrams  Unit = 2
	UnitOunces Unit = 5
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

type AutoOffSetting uint8

const (
	AutoOffDisabled AutoOffSetting = 0
	AutoOff5Min     AutoOffSetting = 1
	AutoOff10Min    AutoOffSetting = 2
	AutoOff20Min    AutoOffSetting = 3
	AutoOff30Min    AutoOffSetting = 4
	AutoOff60Min    AutoOffSetting = 5
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

type StatusMessage struct {
	StatusLength       uint8
	Battery            float64
	IsTimerRunning     bool
	Unit               Unit
	IsCountdownRunning bool
	ScaleMode          ScaleMode
	IsTared            bool
	SleepTimerSetting  AutoOffSetting
	KeyDisableSetting  KeyDisableSetting
	SoundSetting       SoundSetting
	ResolutionSetting  ResolutionSetting
	CapacitySetting    CapacitySetting
	TimerValue         uint16
}
