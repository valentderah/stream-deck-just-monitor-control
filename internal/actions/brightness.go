package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
)

const brightnessMax = 100

func defaultBrightnessSettings() levelSettings {
	return levelSettings{Mode: LevelModeSet, Value: 50, Step: 10, ToggleA: 20, ToggleB: 80}
}

var brightnessSchema = levelSchema(defaultBrightnessSettings(), levelModes)

type brightnessIO struct {
	mgr display.Manager
}

func (b brightnessIO) read(ctx context.Context, monitorID string) (uint32, uint32, error) {
	current, err := b.mgr.GetBrightness(ctx, monitorID)
	return current, brightnessMax, err
}

func (b brightnessIO) write(ctx context.Context, monitorID string, raw uint32) error {
	return b.mgr.SetBrightness(ctx, monitorID, raw)
}

func NewBrightness(mgr display.Manager, resp Responder) *Level {
	return newLevel(mgr, resp, brightnessIO{mgr}, defaultBrightnessSettings(), brightnessSchema, levelModes)
}
