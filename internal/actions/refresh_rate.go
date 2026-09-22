package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
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

func (a *RefreshRate) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	var s refreshSettings
	_ = parseSettings(ev, &s)
	if s.MonitorID == "" || s.Numerator == 0 {
		_ = a.resp.ShowAlert(ev.Context)
		return display.ErrNoMonitorsSelected
	}
	den := s.Denominator
	if den == 0 {
		den = 1
	}
	rate := display.RefreshRate{Numerator: s.Numerator, Denominator: den}
	if err := a.mgr.SetRefreshRate(ctx, s.MonitorID, rate); err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}
	_ = a.resp.ShowOk(ev.Context)
	return nil
}

func (a *RefreshRate) OnWillAppear(context.Context, streamdeck.Event) error { return nil }

func (a *RefreshRate) OnPropertyInspectorDidAppear(ctx context.Context, ev streamdeck.Event) error {
	return sendMonitorsPayload(ctx, a.mgr, a.resp, ev)
}

func (a *RefreshRate) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	if HandleCommonPluginMessage(ctx, a.mgr, a.resp, ev) {
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
