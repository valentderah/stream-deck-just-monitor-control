package actions

import "github.com/valentderah/stream-deck-just-monitor-control/internal/display"

const vcpContrast byte = 0x12

func defaultContrastSettings() levelSettings {
	return levelSettings{Mode: LevelModeSet, Value: 50, Step: 5, ToggleA: 50, ToggleB: 75}
}

func defaultContrastDialSettings() levelSettings {
	return levelSettings{Mode: LevelModeToggle, Value: 50, Step: 5, ToggleA: 50, ToggleB: 75}
}

var (
	contrastKey  = keyProfile(defaultContrastSettings(), levelModes)
	contrastDial = dialProfile(defaultContrastDialSettings(), levelDialModes)
)

func NewContrast(mgr display.Manager, resp Responder) *Level {
	return newLevel(mgr, resp, vcpIO{mgr, vcpContrast}, contrastKey, contrastDial)
}
