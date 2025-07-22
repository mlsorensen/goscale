package decode

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/mlsorensen/goscale/pkg/scales/lunar/comms/encode"
)

type LunarMessage interface{}

type UnhandledMessage struct {
	CommandID byte
	MsgType   *byte // Can be nil if not a nested message
	Payload   []byte
	RawFrame  []byte // Add this field
}

// DecodeNotification decodes messages coming from the Lunar
// It assumes the 'data' buffer contains one complete message frame.
func DecodeNotification(data []byte) (LunarMessage, error) {
	// 1. Find the start of a message (EF DD)
	idx := bytes.Index(data, []byte{encode.HeaderPrefix1, encode.HeaderPrefix2})
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
