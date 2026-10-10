package actions

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
)

func TestSleepSchemaMatchesSettings(t *testing.T) {
	if err := inspector.Verify(sleepSchema, defaultSleepSettings()); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeSleepSettings(t *testing.T) {
	got, err := decodeSettings(settingsEvent(`{"mode":"","asleep":true}`), sleepSchema, defaultSleepSettings())
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != SleepModeSleep || !got.Asleep {
		t.Fatalf("got %+v", got)
	}
}

func TestSleepToggleSavesStateAfterSuccess(t *testing.T) {
	mgr := &fakeManager{}
	resp := &fakeResponder{}
	if err := NewSleep(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"toggle"}`)); err != nil {
		t.Fatal(err)
	}
	if mgr.sleeps != 1 || mgr.wakes != 0 || resp.oks != 1 {
		t.Fatalf("sleeps=%d wakes=%d oks=%d, want 1, 0, 1", mgr.sleeps, mgr.wakes, resp.oks)
	}
	if len(resp.settings) != 1 || !resp.settings[0].(sleepSettings).Asleep {
		t.Fatalf("saved settings = %v, want one save with asleep=true", resp.settings)
	}
	if !reflect.DeepEqual(resp.states, []int{sleepStateAsleep}) {
		t.Fatalf("states=%v, want [1]", resp.states)
	}
}

func TestSleepFailureAlertsAndKeepsState(t *testing.T) {
	mgr := &fakeManager{sleepErr: errBoom}
	resp := &fakeResponder{}
	err := NewSleep(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"toggle"}`))
	if !errors.Is(err, errBoom) || resp.alerts != 1 || resp.oks != 0 {
		t.Fatalf("err=%v alerts=%d oks=%d, want errBoom, 1, 0", err, resp.alerts, resp.oks)
	}
	if len(resp.settings) != 0 || len(resp.states) != 0 {
		t.Fatalf("settings=%v states=%v, want nothing saved", resp.settings, resp.states)
	}
}

func TestSleepToggleWakesWhenAsleep(t *testing.T) {
	mgr := &fakeManager{}
	resp := &fakeResponder{}
	if err := NewSleep(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"toggle","asleep":true}`)); err != nil {
		t.Fatal(err)
	}
	if mgr.sleeps != 0 || mgr.wakes != 1 || resp.settings[0].(sleepSettings).Asleep {
		t.Fatalf("sleeps=%d wakes=%d settings=%v, want a wake and asleep=false", mgr.sleeps, mgr.wakes, resp.settings)
	}
}

func TestSleepRejectsInvalidMode(t *testing.T) {
	resp := &fakeResponder{}
	err := NewSleep(&fakeManager{}, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"bogus"}`))
	if !errors.Is(err, errInvalidMode) || resp.alerts != 1 {
		t.Fatalf("got %v with %d alerts, want errInvalidMode and 1 alert", err, resp.alerts)
	}
}
