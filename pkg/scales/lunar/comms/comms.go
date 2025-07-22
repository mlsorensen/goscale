// Package comms provides communication details for Lunar
package comms

import "github.com/mlsorensen/goscale/pkg/scales/lunar/comms/encode"

import "tinygo.org/x/bluetooth"

var (
	LunarServiceUUID, _     = bluetooth.ParseUUID("49535343-fe7d-4ae5-8fa9-9fafd205e455")
	LunarCommandCharUUID, _ = bluetooth.ParseUUID("49535343-8841-43f4-a8d4-ecbe34729bb3")
	LunarNotifyCharUUID, _  = bluetooth.ParseUUID("49535343-1e4d-4bd9-ba61-23c647249616")
)

var (
	IdentifyCommand            = encode.BuildIdentifyCommand()
	NotificationRequestCommand = encode.BuildNotificationRequestCommand()
	TareCommand                = encode.BuildTareCommand()
	GetStatusCommand           = encode.BuildGetStatusCommand()
)
