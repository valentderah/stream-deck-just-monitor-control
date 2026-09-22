package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

type RawVCP struct {
	mgr  display.Manager
	resp Responder
}

func NewRawVCP(mgr display.Manager, resp Responder) *RawVCP {
	return &RawVCP{mgr: mgr, resp: resp}
}

type rawVCPSettings struct {
	Code      uint32 `json:"code"`
	Value     uint32 `json:"value"`
	MonitorID string `json:"monitorId"`
}

func (a *RawVCP) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	var s rawVCPSettings
	_ = parseSettings(ev, &s)
	if s.MonitorID == "" {
		_ = a.resp.ShowAlert(ev.Context)
		return display.ErrNoMonitorsSelected
	}
	if err := a.mgr.SetVCP(ctx, s.MonitorID, byte(s.Code), s.Value); err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}
	_ = a.resp.ShowOk(ev.Context)
	return nil
}

func (a *RawVCP) OnWillAppear(context.Context, streamdeck.Event) error { return nil }

func (a *RawVCP) OnPropertyInspectorDidAppear(ctx context.Context, ev streamdeck.Event) error {
	return sendMonitorsPayload(ctx, a.mgr, a.resp, ev)
}

func (a *RawVCP) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	_ = HandleCommonPluginMessage(ctx, a.mgr, a.resp, ev)
	return nil
}
