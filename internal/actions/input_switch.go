package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

type InputSwitch struct {
	mgr  display.Manager
	resp Responder
}

func NewInputSwitch(mgr display.Manager, resp Responder) *InputSwitch {
	return &InputSwitch{mgr: mgr, resp: resp}
}

type inputSettings struct {
	Mode      string `json:"mode"`
	Port      uint32 `json:"port"`
	PortA     uint32 `json:"portA"`
	PortB     uint32 `json:"portB"`
	MonitorID string `json:"monitorId"`
}

func nextTogglePort(current, a, b uint32) uint32 {
	if current == a {
		return b
	}
	return a
}

func (a *InputSwitch) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	var s inputSettings
	_ = parseSettings(ev, &s)
	if s.MonitorID == "" {
		_ = a.resp.ShowAlert(ev.Context)
		return display.ErrNoMonitorsSelected
	}

	var target uint32
	var targetState int

	switch s.Mode {
	case "toggle":
		cur, err := a.mgr.GetInputSource(ctx, s.MonitorID)
		if err != nil {
			cur = s.PortA
		}
		target = nextTogglePort(cur, s.PortA, s.PortB)
		if target == s.PortB {
			targetState = 1
		} else {
			targetState = 0
		}
	default:
		target = s.Port
		targetState = 0
	}

	if err := a.mgr.SetInputSource(ctx, s.MonitorID, target); err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}

	if s.Mode == "toggle" {
		_ = a.resp.SetState(ev.Context, targetState)
	}

	_ = a.resp.ShowOk(ev.Context)
	return nil
}

func (a *InputSwitch) OnWillAppear(ctx context.Context, ev streamdeck.Event) error {
	var s inputSettings
	_ = parseSettings(ev, &s)
	if s.MonitorID == "" || s.Mode != "toggle" {
		return nil
	}

	if cur, err := a.mgr.GetInputSource(ctx, s.MonitorID); err == nil {
		if cur == s.PortB {
			_ = a.resp.SetState(ev.Context, 1)
		} else {
			_ = a.resp.SetState(ev.Context, 0)
		}
	}
	return nil
}

func (a *InputSwitch) OnPropertyInspectorDidAppear(ctx context.Context, ev streamdeck.Event) error {
	return sendMonitorsPayload(ctx, a.mgr, a.resp, ev)
}

func (a *InputSwitch) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	_ = HandleCommonPluginMessage(ctx, a.mgr, a.resp, ev)
	return nil
}
