package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

const (
	sleepStateAwake  = 0
	sleepStateAsleep = 1
)

type Sleep struct {
	inspectorHost
}

func NewSleep(mgr display.Manager, resp Responder) *Sleep {
	return &Sleep{inspectorHost{mgr: mgr, resp: resp, schema: sleepSchema}}
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

func sleepState(asleep bool) int {
	if asleep {
		return sleepStateAsleep
	}
	return sleepStateAwake
}

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

	var sleep bool
	switch s.Mode {
	case SleepModeSleep:
		sleep = true
	case SleepModeWake:
		sleep = false
	case SleepModeToggle:
		sleep = !s.Asleep
	default:
		_ = a.resp.ShowAlert(ev.Context)
		return errInvalidMode
	}

	if sleep {
		err = a.mgr.Sleep(ctx, s.MonitorIDs)
	} else {
		err = a.mgr.Wake(ctx, s.MonitorIDs)
	}
	if err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}

	// The toggle state is saved only after the monitors followed, so a failed press can be repeated.
	if s.Mode == SleepModeToggle {
		s.Asleep = sleep
		_ = a.resp.SetSettings(ev.Context, s)
	}
	_ = a.resp.SetState(ev.Context, sleepState(sleep))
	_ = a.resp.ShowOk(ev.Context)
	return nil
}

func (a *Sleep) OnWillAppear(ctx context.Context, ev streamdeck.Event) error {
	s, _ := decodeSettings(ev, sleepSchema, defaultSleepSettings())
	return a.resp.SetState(ev.Context, sleepState(s.Asleep))
}
