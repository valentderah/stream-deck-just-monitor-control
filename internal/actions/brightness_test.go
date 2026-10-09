package actions

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
)

func TestBrightnessSchemaMatchesSettings(t *testing.T) {
	if err := inspector.Verify(brightnessSchema, defaultBrightnessSettings()); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeBrightnessSettings(t *testing.T) {
	cases := []struct {
		name     string
		settings string
		want     levelSettings
	}{
		{"missing keys keep defaults", `{}`,
			levelSettings{Mode: LevelModeSet, Value: 50, Step: 10, ToggleA: 20, ToggleB: 80}},
		{"zero step and empty mode become defaults", `{"mode":"","step":0}`,
			levelSettings{Mode: LevelModeSet, Value: 50, Step: 10, ToggleA: 20, ToggleB: 80}},
		{"zero levels are kept", `{"value":0,"toggleA":0,"toggleB":0}`,
			levelSettings{Mode: LevelModeSet, Value: 0, Step: 10, ToggleA: 0, ToggleB: 0}},
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

func TestBrightnessModes(t *testing.T) {
	cases := []struct {
		settings string
		current  uint32
		want     uint32
	}{
		{`{"monitorIds":["m"],"mode":"set"}`, 10, 50},
		{`{"monitorIds":["m"],"mode":"step"}`, 50, 60},
		{`{"monitorIds":["m"],"mode":"step"}`, 95, 100},
		{`{"monitorIds":["m"],"mode":"toggle"}`, 20, 80},
		{`{"monitorIds":["m"],"mode":"toggle"}`, 80, 20},
	}
	for _, c := range cases {
		mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", fakeBrightnessCode}: {c.current, 100}}}
		if err := NewBrightness(mgr, &fakeResponder{}).OnKeyUp(context.Background(), settingsEvent(c.settings)); err != nil {
			t.Fatalf("%s from %d: %v", c.settings, c.current, err)
		}
		if got := mgr.level("m", fakeBrightnessCode); got != c.want {
			t.Errorf("%s from %d: wrote %d, want %d", c.settings, c.current, got, c.want)
		}
	}
}

func TestBrightnessShowsPercentTitle(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", fakeBrightnessCode}: {50, 100}}}
	resp := &fakeResponder{}
	if err := NewBrightness(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"step"}`)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resp.titles, []string{"60%"}) || resp.oks != 1 {
		t.Fatalf("titles=%v oks=%d, want [60%%] and 1", resp.titles, resp.oks)
	}
}

// Brightness has a fixed range, so Set still works on monitors whose VCP reads fail.
func TestBrightnessSetWritesWhenReadFails(t *testing.T) {
	mgr := &fakeManager{readErr: map[string]error{"m": errBoom}}
	resp := &fakeResponder{}
	if err := NewBrightness(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"set","value":70}`)); err != nil {
		t.Fatal(err)
	}
	if got := mgr.level("m", fakeBrightnessCode); got != 70 || resp.alerts != 0 {
		t.Fatalf("wrote %d with %d alerts, want 70 and 0", got, resp.alerts)
	}
}

func TestBrightnessStepAndToggleAlertWhenReadFails(t *testing.T) {
	for _, mode := range []string{"step", "toggle"} {
		mgr := &fakeManager{
			levels:  map[fakeKey]fakeLevel{{"m", fakeBrightnessCode}: {10, 100}},
			readErr: map[string]error{"m": errBoom},
		}
		resp := &fakeResponder{}
		err := NewBrightness(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"`+mode+`"}`))
		if !errors.Is(err, errBoom) || resp.alerts != 1 {
			t.Errorf("%s: err=%v alerts=%d, want errBoom and 1", mode, err, resp.alerts)
		}
		if got := mgr.level("m", fakeBrightnessCode); got != 10 {
			t.Errorf("%s: level is %d, want 10 (no write)", mode, got)
		}
	}
}

func TestBrightnessRejectsInvalidMode(t *testing.T) {
	for _, mode := range []string{"bogus", "mute"} {
		resp := &fakeResponder{}
		err := NewBrightness(&fakeManager{}, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"`+mode+`"}`))
		if !errors.Is(err, errInvalidMode) || resp.alerts != 1 {
			t.Fatalf("%s: got %v with %d alerts, want errInvalidMode and 1 alert", mode, err, resp.alerts)
		}
	}
}
