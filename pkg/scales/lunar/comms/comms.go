// package comms handles the low-level data packing for Acaia scale communication.
package comms

// Constants for the Acaia communication protocol.
const (
	// HeaderPrefix1 is the first byte of the message header.
	HeaderPrefix1 byte = 0xEF
	// HeaderPrefix2 is the second byte of the message header.
	HeaderPrefix2 byte = 0xDD
)

// PackData is the new, single, correct packer, based on your working 'encode' function.
func PackData(messageType byte, payload []byte) []byte {
	// Start with the required 3-byte header
	message := []byte{HeaderPrefix1, HeaderPrefix2, messageType}

	// Append the entire payload
	message = append(message, payload...)

	// Calculate the split checksum based on the payload only
	var csum1, csum2 byte
	for i, b := range payload {
		if i%2 == 0 {
			csum1 += b
		} else {
			csum2 += b
		}
	}

	// Append the two checksum bytes
	message = append(message, csum1)
	message = append(message, csum2)

	return message
}

// BuildIdentifyCommand now uses the CORRECT payload.
func BuildIdentifyCommand() []byte {
	const cmdIdentify byte = 11 // 0x0B

	// This is the correct, hardcoded "App ID" payload.
	payload := []byte{
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
		0x38, 0x39, 0x30, 0x31, 0x32, 0x33, 0x34,
	}

	// Use the one true packer
	return PackData(cmdIdentify, payload)
}

// BuildNotificationRequestCommand now uses the CORRECT payload with the length prefix.
func BuildNotificationRequestCommand() []byte {
	const cmdEventRequest byte = 12 // 0x0C

	// The event data we want to subscribe to.
	eventData := []byte{
		0, // weight
		1, // weight argument
		1, // battery
		2, // battery argument
		2, // timer
		5, // timer argument
		3, // key
		4, // setting
	}

	// Create the final payload by prefixing the event data with its own length + 1.
	// len(eventData) is 8. So the prefix byte is 8+1=9.
	payload := make([]byte, 1+len(eventData))
	payload[0] = byte(len(eventData) + 1)
	copy(payload[1:], eventData)

	// Use the one true packer
	return PackData(cmdEventRequest, payload)
}

// BuildTareCommand creates the command to tare the scale.
func BuildTareCommand() []byte {
	// The command ID for "key press" or "timer actions" is typically 4 (0x04).
	const cmdKeyAction byte = 4

	// The payload for a tare action is [0x00, 0x00].
	// The first byte indicates the action type (e.g., key press),
	// and the second is the ID for "tare".
	payload := []byte{0x00}

	// Use the one, true packer function that you confirmed works.
	return PackData(cmdKeyAction, payload)
}

// BuildGetStatusCommand creates the command to request a single status update from the scale.
// This is often used as a simple heartbeat if a more complex one isn't required.
func BuildGetStatusCommand() []byte {
	// The command ID for "get status" is 11 (0x0B).
	const cmdGetStatus byte = 11

	// The Java `app_command` function specifies a payload of a single byte with value 0.
	payload := []byte{0x00}

	// Use the one, true packer function.
	return PackData(cmdGetStatus, payload)
}
