package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
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

func defaultRawVCPSettings() rawVCPSettings {
	return rawVCPSettings{Code: 16, Value: 50}
}

var rawVCPSchema = inspector.MustBuild(defaultRawVCPSettings(),
	inspector.MonitorField(),
	inspector.Number("code", "VCPCode").WithRange(0, 255),
	inspector.Number("value", "VCPValue").WithRange(0, 65535),
)

func (a *RawVCP) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	s, err := decodeSettings(ev, rawVCPSchema, defaultRawVCPSettings())
	if err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}
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

func (a *RawVCP) OnPropertyInspectorDidAppear(context.Context, streamdeck.Event) error {
	return nil
}

func (a *RawVCP) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	_ = HandleCommonPluginMessage(ctx, a.mgr, a.resp, ev, rawVCPSchema)
	return nil
}
