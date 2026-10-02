package actions

import (
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
)

func TestRefreshSchemaMatchesSettings(t *testing.T) {
	if err := inspector.Verify(refreshSchema, defaultRefreshSettings()); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeRefreshSettingsDenominator(t *testing.T) {
	for _, settings := range []string{`{"numerator":60,"denominator":0}`, `{"numerator":60}`} {
		got, err := decodeSettings(settingsEvent(settings), refreshSchema, defaultRefreshSettings())
		if err != nil {
			t.Fatal(err)
		}
		if got.Numerator != 60 || got.Denominator != 1 {
			t.Errorf("%s: got %+v", settings, got)
		}
	}
}
