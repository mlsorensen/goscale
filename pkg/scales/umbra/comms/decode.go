package comms

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// DecodeNotification decodes messages coming from the Umbra. Frame structure
// is identical to Lunar; only the weight payload byte order differs.
func DecodeNotification(data []byte) (UmbraMessage, error) {
	idx := bytes.Index(data, []byte{HeaderPrefix1, HeaderPrefix2})
	if idx == -1 {
		return nil, errors.New("message header not found")
	}
	frame := data[idx:]

	if len(frame) < 4 {
		return nil, errors.New("incomplete message frame: too short for header and length")
	}

	payloadLen := int(frame[3])
	expectedFrameLen := payloadLen + 5
	if len(frame) < expectedFrameLen {
		return nil, fmt.Errorf("message frame length mismatch: expected %d bytes, but buffer only has %d", expectedFrameLen, len(frame))
	}

	frame = frame[:expectedFrameLen]
	commandID := frame[2]

	switch commandID {
	case 12:
		msgType := frame[4]
		payload := frame[5 : len(frame)-2]
		return decodeEventMessage(msgType, payload, frame)

	case 8:
		payload := frame[3 : len(frame)-2]
		return DecodeStatusMessage(payload)

	case 7:
		payload := frame[3 : len(frame)-2]
		return DecodeDeviceInfoMessage(payload)

	default:
		return UnhandledMessage{
			CommandID: commandID,
			MsgType:   nil,
			Payload:   frame[4 : len(frame)-2],
			RawFrame:  frame,
		}, nil
	}
}

func decodeEventMessage(msgType byte, payload []byte, rawFrame []byte) (UmbraMessage, error) {
	switch msgType {
	case 5:
		msg, err := decodeWeight(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to decode weight for msgType 5: %w", err)
		}
		return msg, nil

	default:
		return UnhandledMessage{
			CommandID: 12,
			MsgType:   &msgType,
			Payload:   payload,
			RawFrame:  rawFrame,
		}, nil
	}
}

// decodeWeight parses the 6-byte weight event payload.
//
// The Umbra reports its 4-byte raw value big-endian (Lunar uses little-endian).
// We try big-endian first and fall back to little-endian if the result is
// outside a sane range, matching the Apollo Python driver behaviour.
func decodeWeight(payload []byte) (WeightMessage, error) {
	if len(payload) < 6 {
		return WeightMessage{}, errors.New("weight payload too short")
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
		divisor = 10.0
	}

	isStable := (payload[5] & 0x01) == 0
	sign := 1.0
	if (payload[5] & 0x02) != 0 {
		sign = -1.0
	}
	weightType := WeightType(payload[5] >> 2)

	rawBE := binary.BigEndian.Uint32(payload[0:4])
	weight := sign * (float64(rawBE) / divisor)

	// Sanity check — fall back to little-endian if BE produced an absurd value.
	// 2 kg covers the largest Acaia capacity setting with headroom.
	if weight < -2000 || weight > 2000 {
		rawLE := binary.LittleEndian.Uint32(payload[0:4])
		weight = sign * (float64(rawLE) / divisor)
	}

	return WeightMessage{
		Weight:   weight,
		Type:     weightType,
		IsStable: isStable,
	}, nil
}

func DecodeStatusMessage(payload []byte) (StatusMessage, error) {
	if len(payload) < 9 {
		return StatusMessage{}, fmt.Errorf("invalid payload length: expected at least 9, got %d", len(payload))
	}

	msg := StatusMessage{}
	msg.StatusLength = payload[0]
	msg.Battery = float64(payload[1] & 0x7F)
	msg.IsTimerRunning = (payload[1]>>7)&0x01 == 1
	msg.Unit = Unit(payload[2] & 0x7F)
	msg.IsCountdownRunning = (payload[2]>>7)&0x01 == 1
	msg.ScaleMode = ScaleMode(payload[3] & 0x7F)
	msg.IsTared = (payload[3]>>7)&0x01 == 1
	msg.SleepTimerSetting = AutoOffSetting(payload[4])
	msg.KeyDisableSetting = KeyDisableSetting(payload[5])
	msg.SoundSetting = SoundSetting(payload[6])
	msg.ResolutionSetting = ResolutionSetting(payload[7] ^ 1)
	msg.CapacitySetting = CapacitySetting(payload[8])

	return msg, nil
}

func DecodeDeviceInfoMessage(payload []byte) (DeviceInfoMessage, error) {
	if len(payload) != 7 {
		return DeviceInfoMessage{}, fmt.Errorf("invalid payload length for device info: expected 7, got %d", len(payload))
	}

	msg := DeviceInfoMessage{}

	mainVer := bcdToDec(payload[3])
	subVer := bcdToDec(payload[4])
	addVer := bcdToDec(payload[2])

	msg.Firmware = FirmwareVersion{Main: mainVer, Sub: subVer, Add: addVer}
	msg.IsPasswordSet = payload[6] != 0

	return msg, nil
}

func bcdToDec(bcd byte) uint8 {
	return (bcd>>4)*10 + (bcd & 0x0F)
}
