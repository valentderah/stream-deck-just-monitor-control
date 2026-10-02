package actions

import (
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
)

func TestHDRSchemaMatchesSettings(t *testing.T) {
	if err := inspector.Verify(hdrSchema, defaultHDRSettings()); err != nil {
		t.Fatal(err)
	}
}
