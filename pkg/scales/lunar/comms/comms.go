// Package comms provides communication details for Lunar
package comms

import "github.com/mlsorensen/goscale/pkg/scales/lunar/comms/encode"

var (
	IdentifyCommand            = encode.BuildIdentifyCommand()
	NotificationRequestCommand = encode.BuildNotificationRequestCommand()
	TareCommand                = encode.BuildTareCommand()
	GetStatusCommand           = encode.BuildGetStatusCommand()
)
