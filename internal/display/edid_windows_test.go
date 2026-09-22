//go:build windows

package display

import "testing"

func TestExtractModelFromEDID(t *testing.T) {
	edid := make([]byte, 128)
	// Descriptor at offset 54: tag 0xFC (monitor name)
	off := 54
	edid[off+3] = 0xFC
	copy(edid[off+5:], []byte("PORTAL AF24H1\n"))
	got := extractModelFromEDID(edid)
	if got != "PORTAL AF24H1" {
		t.Fatalf("got %q", got)
	}
}
