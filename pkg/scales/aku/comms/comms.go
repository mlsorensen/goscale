package comms

import (
	"tinygo.org/x/bluetooth"
)

var (
	AkuServiceUUID, _     = bluetooth.ParseUUID("FFF0")
	AkuCommandCharUUID, _ = bluetooth.ParseUUID("FFF2")
	AkuNotifyCharUUID, _  = bluetooth.ParseUUID("FFF1")
)

// DecodeStatusUpdate decodes the raw Aku notification. Returns the weight and whether decode was successful
func DecodeStatusUpdate(rawStatus []byte) (float64, bool) {
	if rawStatus[1] == 0x01 {
		sign := 1.0
		if (rawStatus[3] & 0x10) != 0 {
			sign = -1.0
		}
		actualData := sign * (float64(((int(rawStatus[3]) & 0x0f) << 16) + (int(rawStatus[4]) << 8) + int(rawStatus[5])))
		return actualData / 100, true
	}
	return 0, false
}
