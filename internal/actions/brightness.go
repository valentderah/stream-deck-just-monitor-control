package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
)

const brightnessFallbackMax = 100

func defaultBrightnessSettings() levelSettings {
	return levelSettings{Mode: LevelModeSet, Value: 50, Step: 10, ToggleA: 20, ToggleB: 80}
}

func defaultBrightnessDialSettings() levelSettings {
	return levelSettings{Mode: LevelModeToggle, Value: 50, Step: 5, ToggleA: 20, ToggleB: 80}
}

var (
	brightnessKey  = keyProfile(defaultBrightnessSettings(), levelModes)
	brightnessDial = dialProfile(defaultBrightnessDialSettings(), levelDialModes)
)

// brightnessIO assumes the standard 0-100 range when the monitor does not report one,
// so Set still works on monitors whose VCP reads fail.
type brightnessIO struct {
	mgr display.Manager
}

func (b brightnessIO) read(ctx context.Context, monitorID string) (uint32, uint32, error) {
	current, max, err := b.mgr.GetVCP(ctx, monitorID, display.VCPBrightness)
	if err != nil || max == 0 {
		max = brightnessFallbackMax
	}
	return current, max, err
}

func (b brightnessIO) write(ctx context.Context, monitorID string, raw uint32) error {
	return b.mgr.SetVCP(ctx, monitorID, display.VCPBrightness, raw)
}

func NewBrightness(mgr display.Manager, resp Responder) *Level {
	return newLevel(mgr, resp, brightnessIO{mgr}, brightnessKey, brightnessDial)
}
