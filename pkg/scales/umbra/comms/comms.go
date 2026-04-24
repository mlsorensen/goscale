// Package comms provides communication details for the Acaia Lunar Umbra.
package comms

import "tinygo.org/x/bluetooth"

var (
        UmbraServiceUUID, _     = bluetooth.ParseUUID("0000fe40-cc7a-482a-984a-7f2ed5b3e58f")
	UmbraCommandCharUUID, _ = bluetooth.ParseUUID("0000fe41-8e22-4541-9d4c-21edae82ed19")
	UmbraNotifyCharUUID, _  = bluetooth.ParseUUID("0000fe42-8e22-4541-9d4c-21edae82ed19")
)

var (
	IdentifyCommand            = BuildIdentifyCommand()
	NotificationRequestCommand = BuildNotificationRequestCommand()
	TareCommand                = BuildTareCommand()
	GetStatusCommand           = BuildGetStatusCommand()
)
