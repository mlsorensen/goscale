package decode

import "fmt"

// FirmwareVersion holds the main, sub, and additional version numbers for the device firmware.
type FirmwareVersion struct {
	Main uint8
	Sub  uint8
	Add  uint8
}

// String returns a formatted version string, e.g., "1.0.18".
func (v FirmwareVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Main, v.Sub, v.Add)
}

// DeviceInfoMessage holds the parsed device information from a type 7 info event message.
type DeviceInfoMessage struct {
	Firmware      FirmwareVersion
	IsPasswordSet bool
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
