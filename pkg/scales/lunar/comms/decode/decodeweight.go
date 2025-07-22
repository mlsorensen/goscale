package decode

import (
	"encoding/binary"
	"errors"
	"fmt"
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

// decodeWeight parses the 6-byte weight event payload.
func decodeWeight(payload []byte) (WeightMessage, error) {
	if len(payload) < 6 {
		return WeightMessage{}, errors.New("weight payload too short")
	}

	// payload[4] is the divisor (n_dp in the SDK)
	unit := payload[4]
	var divisor float64
	switch unit {
	case 1:
		divisor = 10.0
	case 2:
		divisor = 100.0
	case 3:
		divisor = 1000.0
	case 4:
		divisor = 10000.0
	default:
		divisor = 10.0
	}

	// payload[5] contains packed bitwise flags:
	// Bit 0 (0x01): Stability (0 = stable, 1 = unstable)
	// Bit 1 (0x02): Sign (1 = negative)
	// Bits 2-7    : Weight Type (Net, Gross, etc.)
	isStable := (payload[5] & 0x01) == 0
	sign := 1.0
	if (payload[5] & 0x02) != 0 {
		sign = -1.0
	}
	weightType := WeightType(payload[5] >> 2)

	// payload[0:4] is the raw weight value (n_data)
	raw := binary.LittleEndian.Uint32(payload[0:4])
	weight := sign * (float64(raw) / divisor)

	return WeightMessage{
		Weight:   weight,
		Type:     weightType,
		IsStable: isStable,
	}, nil
}
