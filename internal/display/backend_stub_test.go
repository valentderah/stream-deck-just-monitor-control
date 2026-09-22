//go:build !windows

package display_test

import (
	"context"
	"errors"
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
)

func TestStubGetMonitors(t *testing.T) {
	m := display.NewManager()
	_, err := m.GetMonitors(context.Background())
	if !errors.Is(err, display.ErrUnsupported) {
		t.Fatalf("got %v", err)
	}
}
