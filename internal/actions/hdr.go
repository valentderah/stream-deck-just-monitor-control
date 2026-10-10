package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

type HDR struct {
	inspectorHost
}

func NewHDR(mgr display.Manager, resp Responder) *HDR {
	return &HDR{inspectorHost{mgr: mgr, resp: resp, schema: hdrSchema}}
}

type hdrSettings struct {
	MonitorIDs []string `json:"monitorIds"`
}

func defaultHDRSettings() hdrSettings {
	return hdrSettings{}
}

var hdrSchema = inspector.MustBuild(defaultHDRSettings(), inspector.MonitorsField())

func hdrState(on bool) int {
	if on {
		return 1
	}
	return 0
}

func (a *HDR) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	s, err := decodeSettings(ev, hdrSchema, defaultHDRSettings())
	if err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}
	if len(s.MonitorIDs) == 0 {
		_ = a.resp.ShowAlert(ev.Context)
		return display.ErrNoMonitorsSelected
	}

	// Flip relative to the first monitor. When its state is unreadable, turn HDR on.
	wantOn := true
	if on, err := a.mgr.GetHDR(ctx, s.MonitorIDs[0]); err == nil {
		wantOn = !on
	}

	err = forEachMonitorParallel(s.MonitorIDs, func(id string) error {
		return a.mgr.SetHDR(ctx, id, wantOn)
	})
	if err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}

	_ = a.resp.SetState(ev.Context, hdrState(wantOn))
	_ = a.resp.ShowOk(ev.Context)
	return nil
}

func (a *HDR) OnWillAppear(ctx context.Context, ev streamdeck.Event) error {
	s, _ := decodeSettings(ev, hdrSchema, defaultHDRSettings())
	if len(s.MonitorIDs) == 0 {
		return nil
	}
	on, err := a.mgr.GetHDR(ctx, s.MonitorIDs[0])
	if err != nil {
		return err
	}
	return a.resp.SetState(ev.Context, hdrState(on))
}
