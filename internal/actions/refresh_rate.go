package actions

import (
	"context"
	"errors"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

type RefreshRate struct {
	inspectorHost
}

func NewRefreshRate(mgr display.Manager, resp Responder) *RefreshRate {
	return &RefreshRate{inspectorHost{mgr: mgr, resp: resp, schema: refreshSchema}}
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

func (a *RefreshRate) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	msg := parsePluginMessage(ev)
	if handled, err := handleInspectorMessage(ctx, a.mgr, a.resp, ev, msg, refreshSchema); handled {
		return err
	}
	if msg.Type != "get_refresh_rates" {
		return nil
	}
	// The inspector still gets an answer on failure, so its list stops loading.
	rates, listErr := a.mgr.ListRefreshRates(ctx, msg.MonitorID)
	payload := make([]map[string]any, 0, len(rates))
	for _, r := range rates {
		payload = append(payload, map[string]any{
			"numerator":   r.Numerator,
			"denominator": r.Denominator,
			"hz":          r.Hertz(),
		})
	}
	sendErr := a.resp.SendToPropertyInspector(ev.Context, ev.Action, map[string]any{
		"type":      "refresh_rates",
		"monitorId": msg.MonitorID,
		"rates":     payload,
	})
	return errors.Join(listErr, sendErr)
}
