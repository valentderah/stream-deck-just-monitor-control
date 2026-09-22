package display

import "strings"

func StableMonitorID(deviceID, edidHash string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID != "" && !IsGDIDisplayName(deviceID) {
		return deviceID
	}
	edidHash = strings.TrimSpace(edidHash)
	if edidHash != "" {
		return "edid:" + edidHash
	}
	return ""
}

func IsGDIDisplayName(s string) bool {
	return strings.HasPrefix(s, `\\.\DISPLAY`)
}
