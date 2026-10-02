package actions

import (
	"context"
	"errors"
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

func TestSleepRejectsInvalidMode(t *testing.T) {
	resp := &fakeResponder{}
	err := NewSleep(&fakeManager{}, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"bogus"}`))
	if !errors.Is(err, errInvalidMode) || resp.alerts != 1 {
		t.Fatalf("got %v with %d alerts, want errInvalidMode and 1 alert", err, resp.alerts)
	}
}
