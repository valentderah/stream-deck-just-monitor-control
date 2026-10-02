package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

type RefreshRate struct {
	mgr  display.Manager
	resp Responder
}

func NewRefreshRate(mgr display.Manager, resp Responder) *RefreshRate {
	return &RefreshRate{mgr: mgr, resp: resp}
}

type refreshSettings struct {
	Numerator   uint32 `json:"numerator"`
	Denominator uint32 `json:"denominator"`
	MonitorID   string `json:"monitorId"`
}

func defaultRefreshSettings() refreshSettings {
	return refreshSettings{Denominator: 1}
}

var refreshSchema = inspector.MustBuild(defaultRefreshSettings(),
	inspector.MonitorField(),
	inspector.RefreshRateField(),
)

func (a *RefreshRate) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	s, err := decodeSettings(ev, refreshSchema, defaultRefreshSettings())
	if err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}
	if s.MonitorID == "" || s.Numerator == 0 {
		_ = a.resp.ShowAlert(ev.Context)
		return display.ErrNoMonitorsSelected
	}
	rate := display.RefreshRate{Numerator: s.Numerator, Denominator: s.Denominator}
	if err := a.mgr.SetRefreshRate(ctx, s.MonitorID, rate); err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}
	_ = a.resp.ShowOk(ev.Context)
	return nil
}

func (a *RefreshRate) OnWillAppear(context.Context, streamdeck.Event) error { return nil }

func (a *RefreshRate) OnPropertyInspectorDidAppear(context.Context, streamdeck.Event) error {
	return nil
}

func (a *RefreshRate) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	if HandleCommonPluginMessage(ctx, a.mgr, a.resp, ev, refreshSchema) {
		return nil
	}
	msg, _ := parsePluginMessage(ev)
	if t, _ := msg["type"].(string); t == "get_refresh_rates" {
		id, _ := msg["monitorId"].(string)
		rates, err := a.mgr.ListRefreshRates(ctx, id)
		if err != nil {
			rates = nil
		}
		payload := make([]map[string]any, 0, len(rates))
		for _, r := range rates {
			payload = append(payload, map[string]any{
				"numerator":   r.Numerator,
				"denominator": r.Denominator,
				"hz":          r.Hertz(),
			})
		}
		return a.resp.SendToPropertyInspector(ev.Context, ev.Action, map[string]any{
			"type":      "refresh_rates",
			"monitorId": id,
			"rates":     payload,
		})
	}
	return nil
}
