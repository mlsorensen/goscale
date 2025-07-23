package comms

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// DecodeNotification decodes messages coming from the Lunar
// It assumes the 'data' buffer contains one complete message frame.
func DecodeNotification(data []byte) (LunarMessage, error) {
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

	// 3. Calculate the expected message length from the length byte.
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
		// Pass the original frame down for unhandled packets.
		return decodeEventMessage(msgType, payload, frame)

	case 8: // Settings Message
		payload := frame[3 : len(frame)-2]
		return DecodeStatusMessage(payload) // Placeholder for future implementation

	case 7: // Info Message
		payload := frame[3 : len(frame)-2]
		return DecodeDeviceInfoMessage(payload)

	default:
		// This is an unhandled top-level command.
		return UnhandledMessage{
			CommandID: commandID,
			MsgType:   nil, // Not a nested message
			Payload:   frame[4 : len(frame)-2],
			RawFrame:  frame,
		}, nil
	}
}

// decodeEventMessage handles the inner message layer when the top-level command is 12.
func decodeEventMessage(msgType byte, payload []byte, rawFrame []byte) (LunarMessage, error) {
	switch msgType {
	case 5: // Weight
		msg, err := decodeWeight(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to decode weight for msgType 5: %w", err)
		}
		return msg, nil

	default:
		// This is an unhandled nested message type.
		return UnhandledMessage{
			CommandID: 12, // We know the command was 12
			MsgType:   &msgType,
			Payload:   payload,
			RawFrame:  rawFrame,
		}, nil
	}
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

// DecodeStatusMessage parses the 9-byte or 12-byte payload from a type 8 event message
// and returns a StatusMessage struct.
func DecodeStatusMessage(payload []byte) (StatusMessage, error) {
	if len(payload) < 9 {
		return StatusMessage{}, fmt.Errorf("invalid payload length: expected at least 9, got %d", len(payload))
	}

	msg := StatusMessage{}

	// Byte 0: Status Length
	// The length of the status payload itself.
	msg.StatusLength = payload[0]

	// Byte 1: Battery and Timer Status
	// This byte contains two fields packed using bitwise operations.
	// (7 bits): The battery level (0-100).
	// (1 bit): Indicates if the timer is running.
	msg.Battery = float64(payload[1] & 0x7F)
	msg.IsTimerRunning = (payload[1]>>7)&0x01 == 1

	// Byte 2: Unit and Countdown Status
	// (7 bits): The unit of measurement.
	// (1 bit): Indicates if the countdown is active.
	msg.Unit = Unit(payload[2] & 0x7F)
	msg.IsCountdownRunning = (payload[2]>>7)&0x01 == 1

	// Byte 3: Scale Mode and Tare Status
	// (7 bits): The current mode of the scale.
	// (1 bit): Indicates if the scale is tared.
	msg.ScaleMode = ScaleMode(payload[3] & 0x7F)
	msg.IsTared = (payload[3]>>7)&0x01 == 1

	// Byte 4: Sleep Timer Setting
	msg.SleepTimerSetting = AutoOffSetting(payload[4])

	// Byte 5: Key Disable Setting
	msg.KeyDisableSetting = KeyDisableSetting(payload[5])

	// Byte 6: Sound Setting
	msg.SoundSetting = SoundSetting(payload[6])

	// Byte 7: Resolution Setting
	msg.ResolutionSetting = ResolutionSetting(payload[7] ^ 1)

	// Byte 8: Capacity Setting
	msg.CapacitySetting = CapacitySetting(payload[8])

	return msg, nil
}

// DecodeDeviceInfoMessage parses the 7-byte payload from a type 7 info event
// and returns a DeviceInfoMessage struct.
func DecodeDeviceInfoMessage(payload []byte) (DeviceInfoMessage, error) {
	if len(payload) != 7 {
		return DeviceInfoMessage{}, fmt.Errorf("invalid payload length for device info: expected 7, got %d", len(payload))
	}

	msg := DeviceInfoMessage{}

	// Main Version: payload[3]
	// Sub Version: payload[4]
	// Add Version: payload[2]
	mainVer := bcdToDec(payload[3])
	subVer := bcdToDec(payload[4])
	addVer := bcdToDec(payload[2])

	msg.Firmware = FirmwareVersion{Main: mainVer, Sub: subVer, Add: addVer}
	msg.IsPasswordSet = payload[6] != 0

	return msg, nil
}

// bcdToDec converts a byte in Binary-Coded Decimal format to a standard unsigned integer.
// For example, 0x19 in BCD is treated as the decimal number 19.
func bcdToDec(bcd byte) uint8 {
	return (bcd>>4)*10 + (bcd & 0x0F)
}
