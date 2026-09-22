package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

type HDR struct {
	mgr  display.Manager
	resp Responder
}

func NewHDR(mgr display.Manager, resp Responder) *HDR {
	return &HDR{mgr: mgr, resp: resp}
}

type hdrSettings struct {
	MonitorIDs []string `json:"monitorIds"`
}

func (a *HDR) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	var s hdrSettings
	_ = parseSettings(ev, &s)
	if len(s.MonitorIDs) == 0 {
		_ = a.resp.ShowAlert(ev.Context)
		return display.ErrNoMonitorsSelected
	}

	// Sequential (backend also serializes DisplayConfig); flip relative to first readable state.
	wantOn := true
	if on, err := a.mgr.GetHDR(ctx, s.MonitorIDs[0]); err == nil {
		wantOn = !on
	}

	err := forEachMonitorParallel(s.MonitorIDs, func(id string) error {
		return a.mgr.SetHDR(ctx, id, wantOn)
	})
	if err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}

	if wantOn {
		_ = a.resp.SetState(ev.Context, 1)
	} else {
		_ = a.resp.SetState(ev.Context, 0)
	}
	_ = a.resp.ShowOk(ev.Context)
	return nil
}

func (a *HDR) OnWillAppear(ctx context.Context, ev streamdeck.Event) error {
	var s hdrSettings
	_ = parseSettings(ev, &s)
	if len(s.MonitorIDs) == 0 {
		return nil
	}
	if on, err := a.mgr.GetHDR(ctx, s.MonitorIDs[0]); err == nil {
		if on {
			_ = a.resp.SetState(ev.Context, 1)
		} else {
			_ = a.resp.SetState(ev.Context, 0)
		}
	}
	return nil
}

func (a *HDR) OnPropertyInspectorDidAppear(ctx context.Context, ev streamdeck.Event) error {
	return sendMonitorsPayload(ctx, a.mgr, a.resp, ev)
}

func (a *HDR) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	_ = HandleCommonPluginMessage(ctx, a.mgr, a.resp, ev)
	return nil
}
