package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

type Sleep struct {
	mgr  display.Manager
	resp Responder
}

func NewSleep(mgr display.Manager, resp Responder) *Sleep {
	return &Sleep{mgr: mgr, resp: resp}
}

type sleepSettings struct {
	Mode       string   `json:"mode"`
	Asleep     bool     `json:"asleep"`
	MonitorIDs []string `json:"monitorIds"`
}

func (a *Sleep) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	var s sleepSettings
	_ = parseSettings(ev, &s)
	if len(s.MonitorIDs) == 0 {
		_ = a.resp.ShowAlert(ev.Context)
		return display.ErrNoMonitorsSelected
	}
	mode := s.Mode
	if mode == "" {
		mode = "sleep"
	}
	var err error
	targetState := 0
	switch mode {
	case "wake":
		err = a.mgr.Wake(ctx, s.MonitorIDs)
		s.Asleep = false
		targetState = 0
	case "toggle":
		if s.Asleep {
			err = a.mgr.Wake(ctx, s.MonitorIDs)
			s.Asleep = false
			targetState = 0
		} else {
			err = a.mgr.Sleep(ctx, s.MonitorIDs)
			s.Asleep = true
			targetState = 1
		}
		_ = a.resp.SetSettings(ev.Context, s)
	default:
		err = a.mgr.Sleep(ctx, s.MonitorIDs)
		s.Asleep = true
		targetState = 1
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
	var s sleepSettings
	_ = parseSettings(ev, &s)
	if s.Asleep {
		_ = a.resp.SetState(ev.Context, 1)
	} else {
		_ = a.resp.SetState(ev.Context, 0)
	}
	return nil
}

func (a *Sleep) OnPropertyInspectorDidAppear(ctx context.Context, ev streamdeck.Event) error {
	return sendMonitorsPayload(ctx, a.mgr, a.resp, ev)
}

func (a *Sleep) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	_ = HandleCommonPluginMessage(ctx, a.mgr, a.resp, ev)
	return nil
}
