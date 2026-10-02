package display

import (
	"reflect"
	"testing"
)

func TestParseVCP60FromCapabilities(t *testing.T) {
	caps := `vcp(02 04 05 08 10 12 14 16 60(01 03 0F 11) EE)`
	vals, err := ParseVCPValues(caps, 0x60)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint32{0x01, 0x03, 0x0F, 0x11}
	if !reflect.DeepEqual(vals, want) {
		t.Fatalf("got %v want %v", vals, want)
	}
}
