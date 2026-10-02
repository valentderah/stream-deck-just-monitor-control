package actions

import (
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
)

func TestRawVCPSchemaMatchesSettings(t *testing.T) {
	if err := inspector.Verify(rawVCPSchema, defaultRawVCPSettings()); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeRawVCPSettings(t *testing.T) {
	got, err := decodeSettings(settingsEvent(`{}`), rawVCPSchema, defaultRawVCPSettings())
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != 16 || got.Value != 50 {
		t.Fatalf("defaults: got %+v", got)
	}
	got, err = decodeSettings(settingsEvent(`{"code":0,"value":0}`), rawVCPSchema, defaultRawVCPSettings())
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != 0 || got.Value != 0 {
		t.Fatalf("zeros: got %+v", got)
	}
}
