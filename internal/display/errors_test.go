package display_test

import (
	"errors"
	"math"
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
)

func TestErrUnsupportedIsDistinct(t *testing.T) {
	if display.ErrUnsupported == nil {
		t.Fatal("ErrUnsupported must be set")
	}
	if !errors.Is(display.ErrUnsupported, display.ErrUnsupported) {
		t.Fatal("errors.Is should match")
	}
}

func TestRefreshRateHertz(t *testing.T) {
	r := display.RefreshRate{Numerator: 60000, Denominator: 1001}
	got := r.Hertz()
	want := 60000.0 / 1001.0
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("got %v want %v", got, want)
	}
	zeroDen := display.RefreshRate{Numerator: 60, Denominator: 0}
	if zeroDen.Hertz() != 60 {
		t.Fatalf("zero denominator: got %v", zeroDen.Hertz())
	}
}
