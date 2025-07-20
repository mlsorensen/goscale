package comms

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// AcaiaPacket is the interface for all decoded packets.
type AcaiaPacket interface {
	isAcaiaPacket()
}

// --- Specific Packet Types ---

// WeightPacket represents a weight update (msgType 5).
type WeightPacket struct {
	Weight float64
}

func (p WeightPacket) isAcaiaPacket() {}

// TimerPacket represents a timer update (msgType 7).
type TimerPacket struct {
	Time float64 // Total seconds
}

func (p TimerPacket) isAcaiaPacket() {}

// ButtonPressPacket represents a button event (msgType 8).
type ButtonPressPacket struct {
	Button string   // "tare", "start", "stop", "reset"
	Weight *float64 // Optional weight associated with the press
	Time   *float64 // Optional time associated with the press
}

func (p ButtonPressPacket) isAcaiaPacket() {}

// HeartbeatResponsePacket represents the complex response to a heartbeat (msgType 11).
type HeartbeatResponsePacket struct {
	Weight *float64
	Time   *float64
}

func (p HeartbeatResponsePacket) isAcaiaPacket() {}

// SettingsPacket represents a settings message (command 8).
type SettingsPacket struct {
	// You would define fields here based on the settings payload format
}

func (p SettingsPacket) isAcaiaPacket() {}

// UnhandledPacket represents any message we don't have a specific parser for yet.
type UnhandledPacket struct {
	CommandID byte
	MsgType   *byte // Can be nil if not a nested message
	Payload   []byte
	RawFrame  []byte // Add this field

}

func (p UnhandledPacket) isAcaiaPacket() {}

func decodeWeight(payload []byte) (float64, error) {
	if len(payload) < 6 {
		return 0, errors.New("weight payload too short")
	}

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
		return 0, fmt.Errorf("bad weight unit: %d", unit)
	}

	sign := 1.0
	if (payload[5] & 0x02) != 0 {
		sign = -1.0
	}

	// The Python code tries Big Endian first, then Little Endian.
	// This is unusual. Most devices stick to one. Let's default to Little Endian
	// as it's more common for sensor data, but keep the logic for reference.
	// raw := binary.BigEndian.Uint32(payload[0:4])
	raw := binary.LittleEndian.Uint32(payload[0:4])

	weight := sign * (float64(raw) / divisor)
	return weight, nil
}

func decodeTime(payload []byte) (float64, error) {
	if len(payload) < 3 {
		return 0, errors.New("time payload too short")
	}
	minutes := float64(payload[0])
	seconds := float64(payload[1])
	tenths := float64(payload[2])
	return (minutes * 60) + seconds + (tenths / 10.0), nil
}

// ParseNotification is the main entry point.
// It assumes the 'data' buffer contains one complete message frame.
func ParseNotification(data []byte) (AcaiaPacket, error) {
	// 1. Find the start of a message (EF DD)
	idx := bytes.Index(data, []byte{HeaderPrefix1, HeaderPrefix2})
	if idx == -1 {
		return nil, errors.New("message header not found")
	}
	frame := data[idx:] // Start processing from the header

	// 2. Check for minimal length for header and length byte
	if len(frame) < 4 {
		return nil, errors.New("incomplete message frame: too short for header and length")
	}

	// 3. Calculate expected message length from the length byte.
	// The Python code's calculation is: total_len = payload_len_byte + 5
	payloadLen := int(frame[3])
	expectedFrameLen := payloadLen + 5
	if len(frame) < expectedFrameLen {
		return nil, fmt.Errorf("message frame length mismatch: expected %d bytes, but buffer only has %d", expectedFrameLen, len(frame))
	}

	// We only process the expected length, creating a clean frame.
	frame = frame[:expectedFrameLen]
	commandID := frame[2]

	switch commandID {
	case 12: // Nested Message Type
		// The nested message starts after the commandID (12) and the msgType.
		msgType := frame[4]
		// The payload is between the msgType and the 2-byte checksum.
		payload := frame[5 : len(frame)-2]
		// Pass the original frame down for logging purposes.
		return parseNestedMessage(msgType, payload, frame)

	case 8: // Settings Message
		return SettingsPacket{}, nil // Placeholder for future implementation

	default:
		// This is an unhandled top-level command.
		return UnhandledPacket{
			CommandID: commandID,
			MsgType:   nil, // Not a nested message
			Payload:   frame[4 : len(frame)-2],
			RawFrame:  frame,
		}, nil
	}
}

// parseNestedMessage handles the inner message layer when the top-level command is 12.
// It now correctly returns (AcaiaPacket, error).
func parseNestedMessage(msgType byte, payload []byte, rawFrame []byte) (AcaiaPacket, error) {
	switch msgType {
	case 5: // Weight
		w, err := decodeWeight(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to decode weight for msgType 5: %w", err)
		}
		return WeightPacket{Weight: w}, nil

	case 7: // Timer
		t, err := decodeTime(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to decode time for msgType 7: %w", err)
		}
		return TimerPacket{Time: t}, nil

	case 8: // Button Press
		return parseButtonPress(payload) // This function returns (AcaiaPacket, error)

	case 11: // Heartbeat Response
		var w, t *float64
		// This is a complex payload, check for sub-types
		if len(payload) > 3 && payload[2] == 5 { // Contains weight
			weight, err := decodeWeight(payload[3:])
			if err == nil {
				w = &weight
			}
		}
		if len(payload) > 3 && payload[2] == 7 { // Contains time
			timeVal, err := decodeTime(payload[3:])
			if err == nil {
				t = &timeVal
			}
		}
		return HeartbeatResponsePacket{Weight: w, Time: t}, nil

	default:
		// This is an unhandled nested message type.
		return UnhandledPacket{
			CommandID: 12, // We know the command was 12
			MsgType:   &msgType,
			Payload:   payload,
			RawFrame:  rawFrame,
		}, nil
	}
}

// parseButtonPress now correctly returns (AcaiaPacket, error) for consistency.
func parseButtonPress(payload []byte) (AcaiaPacket, error) {
	if len(payload) < 2 {
		return nil, errors.New("button payload too short")
	}
	p := ButtonPressPacket{}

	switch {
	case payload[0] == 0 && payload[1] == 5:
		p.Button = "tare"
		if len(payload) > 2 {
			w, _ := decodeWeight(payload[2:])
			p.Weight = &w
		}
	case payload[0] == 8 && payload[1] == 5:
		p.Button = "start"
		if len(payload) > 2 {
			w, _ := decodeWeight(payload[2:])
			p.Weight = &w
		}
	case payload[0] == 10 && payload[1] == 7:
		p.Button = "stop"
		if len(payload) > 2 {
			t, _ := decodeTime(payload[2:])
			p.Time = &t
		}
		if len(payload) > 6 {
			w, _ := decodeWeight(payload[6:])
			p.Weight = &w
		}
	case payload[0] == 9 && payload[1] == 7:
		p.Button = "reset"
		if len(payload) > 2 {
			t, _ := decodeTime(payload[2:])
			p.Time = &t
		}
		if len(payload) > 6 {
			w, _ := decodeWeight(payload[6:])
			p.Weight = &w
		}
	default:
		p.Button = "unknown"
	}
	return p, nil
}
