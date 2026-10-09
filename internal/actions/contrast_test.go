package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
)

func TestContrastSchemaMatchesSettings(t *testing.T) {
	if err := inspector.Verify(contrastSchema, defaultContrastSettings()); err != nil {
		t.Fatal(err)
	}
}

func TestContrastDefaults(t *testing.T) {
	got, err := decodeSettings(settingsEvent(`{}`), contrastSchema, defaultContrastSettings())
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != LevelModeSet || got.Value != 50 || got.Step != 5 || got.ToggleA != 50 || got.ToggleB != 75 {
		t.Fatalf("got %+v", got)
	}
}

func TestContrastWritesVCP12(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", 0x12}: {10, 255}}}
	if err := NewContrast(mgr, &fakeResponder{}).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"set","value":50}`)); err != nil {
		t.Fatal(err)
	}
	if got := mgr.level("m", 0x12); got != 128 {
		t.Fatalf("wrote %d, want 128", got)
	}
}

func TestContrastUnsupportedMonitorAlerts(t *testing.T) {
	resp := &fakeResponder{}
	err := NewContrast(&fakeManager{}, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"set","value":50}`))
	if err == nil || resp.alerts != 1 || resp.oks != 0 {
		t.Fatalf("err=%v alerts=%d oks=%d, want an error, 1, 0", err, resp.alerts, resp.oks)
	}
}

func TestContrastRejectsMute(t *testing.T) {
	resp := &fakeResponder{}
	err := NewContrast(&fakeManager{}, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"mute"}`))
	if !errors.Is(err, errInvalidMode) || resp.alerts != 1 {
		t.Fatalf("got %v with %d alerts, want errInvalidMode and 1", err, resp.alerts)
	}
}
