//go:build windows

package display

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

func extractModelFromEDID(edid []byte) string {
	if len(edid) < 128 {
		return ""
	}
	offsets := []int{54, 72, 90, 108}
	for _, off := range offsets {
		if off+18 > len(edid) {
			break
		}
		desc := edid[off : off+18]
		if desc[0] == 0 && desc[1] == 0 && desc[2] == 0 && desc[3] == 0xFC {
			name := strings.TrimSpace(string(desc[5:]))
			name = strings.ReplaceAll(name, "\n", "")
			name = strings.ReplaceAll(name, "\r", "")
			name = strings.Trim(name, "\x00 ")
			if len(name) > 0 {
				return name
			}
		}
	}
	return ""
}

func queryEDIDModel(deviceID string) string {
	if deviceID == "" {
		return ""
	}
	if name := readEDIDModelAt(`SYSTEM\CurrentControlSet\Enum\` + deviceID + `\Device Parameters`); name != "" {
		return name
	}
	// MONITOR\SKG2409\{guid}\0002 → DISPLAY\SKG2409\<instance>\Device Parameters
	parts := strings.Split(deviceID, `\`)
	if len(parts) < 2 {
		return ""
	}
	vendor := parts[1]
	base := `SYSTEM\CurrentControlSet\Enum\DISPLAY\` + vendor
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, base, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return ""
	}
	defer k.Close()
	names, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return ""
	}
	for _, inst := range names {
		if name := readEDIDModelAt(base + `\` + inst + `\Device Parameters`); name != "" {
			return name
		}
	}
	return ""
}

func readEDIDModelAt(path string) string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	edid, _, err := k.GetBinaryValue("EDID")
	if err != nil {
		return ""
	}
	return extractModelFromEDID(edid)
}
