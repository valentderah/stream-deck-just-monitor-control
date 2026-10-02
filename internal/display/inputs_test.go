package display

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type inputNameCase struct {
	Code uint32 `json:"code"`
	Name string `json:"name"`
}

func loadInputNameFixture(t *testing.T) (known, unknown []inputNameCase) {
	t.Helper()
	data, err := os.ReadFile("../../testdata/input-names.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Known   []inputNameCase `json:"known"`
		Unknown []inputNameCase `json:"unknown"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture.Known, fixture.Unknown
}

func TestInputNameMatchesFixture(t *testing.T) {
	known, unknown := loadInputNameFixture(t)
	for _, c := range append(known, unknown...) {
		if got := InputName(c.Code); got != c.Name {
			t.Errorf("InputName(%d) = %q, want %q", c.Code, got, c.Name)
		}
	}
}

func TestKnownInputPortsOrder(t *testing.T) {
	var codes []uint32
	for _, p := range KnownInputPorts() {
		codes = append(codes, p.Code)
	}
	want := []uint32{0x0F, 0x10, 0x11, 0x12, 0x1B, 0x01, 0x03}
	if !reflect.DeepEqual(codes, want) {
		t.Fatalf("got %v want %v", codes, want)
	}
}

func TestInputPortsFromCapabilitiesKeepsOrderAndDropsDuplicates(t *testing.T) {
	got, ok := InputPortsFromCapabilities(`vcp(10 60(11 0F 11 1B) EE)`)
	if !ok {
		t.Fatal("ok = false")
	}
	want := []InputPort{
		{Code: 0x11, Name: "HDMI 1"},
		{Code: 0x0F, Name: "DisplayPort 1"},
		{Code: 0x1B, Name: "USB-C / Type-C"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestInputPortsFromCapabilitiesFallsBack(t *testing.T) {
	for _, caps := range []string{
		"",
		"vcp(10 12)",
		"vcp(60(0F zz))",
		"vcp(60())",
		"vcp(60(0F",
	} {
		got, ok := InputPortsFromCapabilities(caps)
		if ok {
			t.Errorf("%q: ok = true", caps)
		}
		if !reflect.DeepEqual(got, KnownInputPorts()) {
			t.Errorf("%q: got %v, want fallback", caps, got)
		}
	}
}
