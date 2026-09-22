package actions

import "testing"

func TestNextTogglePort(t *testing.T) {
	if nextTogglePort(0x11, 0x11, 0x0F) != 0x0F {
		t.Fatal()
	}
	if nextTogglePort(0x0F, 0x11, 0x0F) != 0x11 {
		t.Fatal()
	}
}
