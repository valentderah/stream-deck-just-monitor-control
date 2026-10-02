package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

type Sleep struct {
	mgr  display.Manager
	resp Responder
}

func NewSleep(mgr display.Manager, resp Responder) *Sleep {
	return &Sleep{mgr: mgr, resp: resp}
}

type SleepMode string

const (
	SleepModeSleep  SleepMode = "sleep"
	SleepModeWake   SleepMode = "wake"
	SleepModeToggle SleepMode = "toggle"
)

type sleepSettings struct {
	Mode       SleepMode `json:"mode"`
	Asleep     bool      `json:"asleep"`
	MonitorIDs []string  `json:"monitorIds"`
}

func defaultSleepSettings() sleepSettings {
	return sleepSettings{Mode: SleepModeSleep}
}

var sleepSchema = inspector.MustBuild(defaultSleepSettings(),
	inspector.MonitorsField(),
	inspector.Select("mode", "Mode", inspector.TypeString,
		inspector.LocalizedOption(SleepModeSleep, "Sleep"),
		inspector.LocalizedOption(SleepModeWake, "Wake"),
		inspector.LocalizedOption(SleepModeToggle, "Toggle"),
	).WithZeroAsUnset(),
)

func (a *Sleep) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	s, err := decodeSettings(ev, sleepSchema, defaultSleepSettings())
	if err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}
	if len(s.MonitorIDs) == 0 {
		_ = a.resp.ShowAlert(ev.Context)
		return display.ErrNoMonitorsSelected
	}
	targetState := 0
	switch s.Mode {
	case SleepModeWake:
		err = a.mgr.Wake(ctx, s.MonitorIDs)
		s.Asleep = false
	case SleepModeToggle:
		if s.Asleep {
			err = a.mgr.Wake(ctx, s.MonitorIDs)
			s.Asleep = false
		} else {
			err = a.mgr.Sleep(ctx, s.MonitorIDs)
			s.Asleep = true
			targetState = 1
		}
		_ = a.resp.SetSettings(ev.Context, s)
	case SleepModeSleep:
		err = a.mgr.Sleep(ctx, s.MonitorIDs)
		s.Asleep = true
		targetState = 1
	default:
		_ = a.resp.ShowAlert(ev.Context)
		return errInvalidMode
	}
	if err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}
	_ = a.resp.SetState(ev.Context, targetState)
	_ = a.resp.ShowOk(ev.Context)
	return nil
}

func (a *Sleep) OnWillAppear(ctx context.Context, ev streamdeck.Event) error {
	s, _ := decodeSettings(ev, sleepSchema, defaultSleepSettings())
	if s.Asleep {
		_ = a.resp.SetState(ev.Context, 1)
	} else {
		_ = a.resp.SetState(ev.Context, 0)
	}
	return nil
}

func (a *Sleep) OnPropertyInspectorDidAppear(context.Context, streamdeck.Event) error {
	return nil
}

func (a *Sleep) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	_ = HandleCommonPluginMessage(ctx, a.mgr, a.resp, ev, sleepSchema)
	return nil
}
