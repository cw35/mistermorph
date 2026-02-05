package telegram

import "strings"

const heartbeatFailureThreshold = 3

func isHeartbeatOK(output string) bool {
	return strings.EqualFold(strings.TrimSpace(output), "HEARTBEAT_OK")
}
