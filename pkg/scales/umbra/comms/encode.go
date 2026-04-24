package comms

// Encode creates an encoded command frame for the Umbra. The framing is the
// same as Lunar — outgoing commands have not changed.
func Encode(messageType byte, payload []byte) []byte {
	message := []byte{HeaderPrefix1, HeaderPrefix2, messageType}
	message = append(message, payload...)

	var csum1, csum2 byte
	for i, b := range payload {
		if i%2 == 0 {
			csum1 += b
		} else {
			csum2 += b
		}
	}

	message = append(message, csum1)
	message = append(message, csum2)

	return message
}

func BuildIdentifyCommand() []byte {
	const cmdIdentify byte = 11
	payload := []byte{
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
		0x38, 0x39, 0x30, 0x31, 0x32, 0x33, 0x34,
	}
	return Encode(cmdIdentify, payload)
}

func BuildNotificationRequestCommand() []byte {
	const cmdEventRequest byte = 12

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

	payload := make([]byte, 1+len(eventData))
	payload[0] = byte(len(eventData) + 1)
	copy(payload[1:], eventData)

	return Encode(cmdEventRequest, payload)
}

func BuildTareCommand() []byte {
	const cmdKeyAction byte = 4
	payload := []byte{0x00}
	return Encode(cmdKeyAction, payload)
}

func BuildGetStatusCommand() []byte {
	const cmdGetStatus byte = 6
	payload := []byte{0x00}
	return Encode(cmdGetStatus, payload)
}

// Setting IDs come from the Acaia SDK's ESETTING_ITEM enum. The Umbra has its
// own setting ID space distinct from the Lunar:
//   e_setting_umbra_sleep = 6
//   e_setting_umbra_beep  = 7
// (cf. AcaiaSettingCommandSpec.java in the official Android SDK)
const (
	settingIDUmbraSleep byte = 6
	settingIDUmbraBeep  byte = 7
)

func BuildAutoOffCommand(setting AutoOffSetting) []byte {
	payload := []byte{0x00, settingIDUmbraSleep, byte(setting)}
	return Encode(10, payload)
}

func BuildSetBeepCommand(beep bool) []byte {
	value := byte(0x00)
	if beep {
		value = 0x01
	}
	payload := []byte{0x00, settingIDUmbraBeep, value}
	return Encode(10, payload)
}
