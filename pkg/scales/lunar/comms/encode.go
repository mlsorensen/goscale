package comms

// Encode creates an encoded message for Lunar
func Encode(messageType byte, payload []byte) []byte {
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

// BuildIdentifyCommand creates the command to identify
func BuildIdentifyCommand() []byte {
	const cmdIdentify byte = 11 // 0x0B
	payload := []byte{
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
		0x38, 0x39, 0x30, 0x31, 0x32, 0x33, 0x34,
	}

	return Encode(cmdIdentify, payload)
}

// BuildNotificationRequestCommand creates a notification request command
func BuildNotificationRequestCommand() []byte {
	const cmdEventRequest byte = 12 // 0x0C

	// The event data we want to subscribe to.
	eventData := []byte{
		0x0,  // weight
		0x01, // weight argument
		0x01, // battery
		0x02, // battery argument
		0x02, // timer
		0x05, // timer argument
		0x03, // key
		0x04, // setting
	}

	// Create the final payload by prefixing the event data with its own length + 1.
	// len(eventData) is 8. So the prefix byte is 8+1=9.
	payload := make([]byte, 1+len(eventData))
	payload[0] = byte(len(eventData) + 1)
	copy(payload[1:], eventData)

	return Encode(cmdEventRequest, payload)
}

// BuildTareCommand creates the command to tare the scale.
func BuildTareCommand() []byte {
	const cmdKeyAction byte = 4
	payload := []byte{0x00}

	return Encode(cmdKeyAction, payload)
}

// BuildGetStatusCommand creates the command to request a single status update from the scale.
// This is often used as a simple heartbeat if a more complex one isn't required.
func BuildGetStatusCommand() []byte {
	const cmdGetStatus byte = 6
	payload := []byte{0x00}

	return Encode(cmdGetStatus, payload)
}

func BuildAutoOffCommand(setting AutoOffSetting) []byte {
	const cmdSetSetting = 10
	payload := []byte{0x00, 0x01, byte(setting)}

	return Encode(10, payload)
}
