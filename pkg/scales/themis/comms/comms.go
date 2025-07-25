package comms

import (
	"log"
	"tinygo.org/x/bluetooth"
)

var (
	ThemisServiceUUID, _     = bluetooth.ParseUUID("0FFE")
	ThemisCommandCharUUID, _ = bluetooth.ParseUUID("FF12")
	ThemisNotifyCharUUID, _  = bluetooth.ParseUUID("FF11")

	ThemisTareCommand = []byte{0x03, 0x0a, 0x01, 0x00, 0x00, 0x08}

	AutoOffSettings = newAutoOffSettingsManager()
)

type StatusUpdate struct {
	ProductNumber    uint8
	Type             uint8
	Milliseconds     uint32  // Combined from bytes 3-5 (indices 2, 3, 4)
	UnitOfWeight     uint8   // BYTE6: Unit of weight (grams only)
	WeightSymbolData uint8   // BYTE7: Weight symbol data points (+/-)
	GramsWeight      float64 // Combined from bytes 8-10 (indices 7, 8, 9) representing grams * 100
	FlowRateSymbol   uint8   // BYTE11: Flow rate symbol data points (+/-)
	FlowRate         float64 // Combined from bytes 12 and 13 (indices 11, 12) representing flow rate * 100
	PowerPercentage  uint8   // BYTE14: Percentage of remaining power
	StandbyTime      uint16  // Combined from bytes 15 and 16 (indices 14, 15) representing standby time in minutes
	BuzzerGear       uint8   // BYTE17: Buzzer gear
	SmoothingSwitch  uint8   // BYTE18: Flow rate smoothing switch
	Reserved1        uint8   // BYTE19: Reserved (00)
	Reserved2        uint8   // BYTE20: Reserved (00)
}

// DecodeStatusUpdate decodes the raw Themis notification. Returns the weight and whether decode was successful
func DecodeStatusUpdate(data []byte) (*StatusUpdate, bool) {
	var n StatusUpdate
	
	if len(data) != 20 {
		return nil, false // Return zeroed struct if data length is incorrect
	}

	// Milliseconds: Combine bytes 3-5 (indices 2, 3, 4) into a uint32 (big-endian)
	n.Milliseconds = uint32(data[2])<<16 | uint32(data[3])<<8 | uint32(data[4])

	// GramsWeight: Combine bytes 8-10 (indices 7, 8, 9) into a uint32 (big-endian) representing grams * 100
	var gramsUint uint32
	gramsUint = uint32(data[7])<<16 | uint32(data[8])<<8 | uint32(data[9])

	// Handle sign based on WeightSymbolData
	if data[6] == 45 { // Check if the value is negative (ASCII for '-')
		n.GramsWeight = -float64(gramsUint) / 100.0
	} else {
		n.GramsWeight = float64(gramsUint) / 100.0
	}

	// FlowRate: Combine bytes 12 and 13 (indices 11, 12) into a uint16 (big-endian) representing flow rate * 100
	var flowRateUint uint16
	flowRateUint = uint16(data[11])<<8 | uint16(data[12])
	n.FlowRate = float64(flowRateUint) / 100.0

	// StandbyTime: Combine bytes 15 and 16 (indices 14, 15) into a uint16 (big-endian) representing minutes
	n.StandbyTime = (uint16(data[14])<<8 | uint16(data[15])) / 10

	// Assign other fields directly from the byte slice
	n.ProductNumber = data[0]
	n.Type = data[1]
	n.UnitOfWeight = data[5]
	n.WeightSymbolData = data[6] // BYTE7: Weight symbol data points
	n.FlowRateSymbol = data[10]  // BYTE11: Flow rate symbol data points
	n.PowerPercentage = data[13] // BYTE14: Percentage of remaining power
	n.BuzzerGear = data[16]      // BYTE17: Buzzer gear
	n.SmoothingSwitch = data[17] // BYTE18: Smoothing switch
	n.Reserved1 = data[18]       // BYTE19: Reserved
	n.Reserved2 = data[19]       // BYTE20: Reserved

	return &n, true
}

func BuildAutoOffCommand(setting AutoOffSetting) []byte {
	payload := []byte{0x03, 0x0a, 0x03, 0x00, uint8(setting)}
	msg := append(payload, CalculateChecksum(payload))
	log.Println(msg)
	return msg
}

func BuildChangeBeepCommand(beep bool) []byte {
	set := 0
	if beep {
		set = 5
	}
	payload := []byte{0x03, 0x0a, 0x02, 0x00, uint8(set)}
	msg := append(payload, CalculateChecksum(payload))
	log.Println(msg)
	return msg
}
