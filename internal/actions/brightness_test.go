package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
)

func TestBrightnessStep(t *testing.T) {
	if got := applyStep(50, 10, 100); got != 60 {
		t.Fatal(got)
	}
	if got := applyStep(95, 10, 100); got != 100 {
		t.Fatal(got)
	}
	if got := applyStep(5, -10, 100); got != 0 {
		t.Fatal(got)
	}
}

func TestBrightnessToggle(t *testing.T) {
	if applyToggle(20, 20, 80) != 80 {
		t.Fatal()
	}
	if applyToggle(80, 20, 80) != 20 {
		t.Fatal()
	}
}

func TestBrightnessSchemaMatchesSettings(t *testing.T) {
	if err := inspector.Verify(brightnessSchema, defaultBrightnessSettings()); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeBrightnessSettings(t *testing.T) {
	cases := []struct {
		name     string
		settings string
		want     brightnessSettings
	}{
		{"missing keys keep defaults", `{}`,
			brightnessSettings{Mode: BrightnessModeSet, Value: 50, Step: 10, ToggleA: 20, ToggleB: 80}},
		{"zero step and empty mode become defaults", `{"mode":"","step":0}`,
			brightnessSettings{Mode: BrightnessModeSet, Value: 50, Step: 10, ToggleA: 20, ToggleB: 80}},
		{"zero levels are kept", `{"value":0,"toggleA":0,"toggleB":0}`,
			brightnessSettings{Mode: BrightnessModeSet, Value: 0, Step: 10, ToggleA: 0, ToggleB: 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := decodeSettings(settingsEvent(c.settings), brightnessSchema, defaultBrightnessSettings())
			if err != nil {
				t.Fatal(err)
			}
			if got.Mode != c.want.Mode || got.Value != c.want.Value || got.Step != c.want.Step ||
				got.ToggleA != c.want.ToggleA || got.ToggleB != c.want.ToggleB {
				t.Fatalf("got %+v want %+v", got, c.want)
			}
		})
	}
}

func TestBrightnessTarget(t *testing.T) {
	s := defaultBrightnessSettings()
	cases := []struct {
		mode    BrightnessMode
		current uint32
		want    uint32
	}{
		{BrightnessModeSet, 10, 50},
		{BrightnessModeStep, 95, 100},
		{BrightnessModeToggle, 20, 80},
	}
	for _, c := range cases {
		s.Mode = c.mode
		got, err := brightnessTarget(s, c.current)
		if err != nil || got != c.want {
			t.Errorf("%s from %d: got %d, %v; want %d", c.mode, c.current, got, err, c.want)
		}
	}
	s.Mode = "bogus"
	if _, err := brightnessTarget(s, 0); !errors.Is(err, errInvalidMode) {
		t.Fatalf("got %v, want errInvalidMode", err)
	}
}

func TestBrightnessRejectsInvalidMode(t *testing.T) {
	resp := &fakeResponder{}
	err := NewBrightness(&fakeManager{}, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"bogus"}`))
	if !errors.Is(err, errInvalidMode) || resp.alerts != 1 {
		t.Fatalf("got %v with %d alerts, want errInvalidMode and 1 alert", err, resp.alerts)
	}
}
