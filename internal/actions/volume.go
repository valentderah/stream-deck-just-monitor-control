package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

const (
	vcpVolume byte = 0x62
	vcpMute   byte = 0x8D

	muteOn  uint32 = 1
	muteOff uint32 = 2

	volumeStateNormal = 0
	volumeStateMuted  = 1
)

var (
	volumeModes     = []LevelMode{LevelModeSet, LevelModeStep, LevelModeToggle, LevelModeMute}
	volumeDialModes = []LevelMode{LevelModeMute, LevelModeSet, LevelModeToggle}
)

func defaultVolumeSettings() levelSettings {
	return levelSettings{Mode: LevelModeSet, Value: 50, Step: 5, ToggleA: 20, ToggleB: 60}
}

func defaultVolumeDialSettings() levelSettings {
	return levelSettings{Mode: LevelModeMute, Value: 50, Step: 2, ToggleA: 20, ToggleB: 60}
}

var (
	volumeKey  = keyProfile(defaultVolumeSettings(), volumeModes)
	volumeDial = dialProfile(defaultVolumeDialSettings(), volumeDialModes)
)

// Volume controls the monitor's own speakers, not the Windows volume.
type Volume struct {
	*Level
}

func NewVolume(mgr display.Manager, resp Responder) *Volume {
	return &Volume{newLevel(mgr, resp, vcpIO{mgr, vcpVolume}, volumeKey, volumeDial)}
}

func muteState(muted bool) int {
	if muted {
		return volumeStateMuted
	}
	return volumeStateNormal
}

func (v *Volume) pressVolume(ctx context.Context, ev streamdeck.Event, onDial bool, p levelProfile) error {
	s, err := v.settings(ev, p)
	if err != nil {
		_ = v.resp.ShowAlert(ev.Context)
		return err
	}
	if s.Mode == LevelModeMute {
		return v.toggleMute(ctx, ev, s.MonitorIDs)
	}
	if err := v.press(ctx, ev, onDial, s); err != nil {
		return err
	}
	_ = v.resp.SetState(ev.Context, volumeStateNormal)
	return nil
}

func (v *Volume) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	return v.pressVolume(ctx, ev, false, v.key)
}

func (v *Volume) OnDialPress(ctx context.Context, ev streamdeck.Event) error {
	return v.pressVolume(ctx, ev, true, v.dial)
}

// toggleMute flips relative to the first monitor and applies the result to all of them.
func (v *Volume) toggleMute(ctx context.Context, ev streamdeck.Event, monitorIDs []string) error {
	current, _, err := v.mgr.GetVCP(ctx, monitorIDs[0], vcpMute)
	if err != nil {
		_ = v.resp.ShowAlert(ev.Context)
		return err
	}
	muted := current == muteOn
	target := muteOn
	if muted {
		target = muteOff
	}

	err = forEachMonitorParallel(monitorIDs, func(id string) error {
		return v.mgr.SetVCP(ctx, id, vcpMute, target)
	})
	if err != nil {
		_ = v.resp.SetState(ev.Context, muteState(muted))
		_ = v.resp.ShowAlert(ev.Context)
		return err
	}

	_ = v.resp.SetState(ev.Context, muteState(!muted))
	_ = v.resp.ShowOk(ev.Context)
	return nil
}

func (v *Volume) OnWillAppear(ctx context.Context, ev streamdeck.Event) error {
	p := v.profile(ev)
	s, _ := decodeSettings(ev, p.schema, p.defaults)
	muted := false
	if s.Mode == LevelModeMute && len(s.MonitorIDs) > 0 {
		if current, _, err := v.mgr.GetVCP(ctx, s.MonitorIDs[0], vcpMute); err == nil {
			muted = current == muteOn
		}
	}
	_ = v.resp.SetState(ev.Context, muteState(muted))
	return v.Level.OnWillAppear(ctx, ev)
}
