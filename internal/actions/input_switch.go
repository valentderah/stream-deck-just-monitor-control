package actions

import (
	"context"
	"errors"
	"time"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

const (
	inputConfirmationAttempts = 6
	inputConfirmationDelay    = 500 * time.Millisecond
)

var errInputSourceNotConfirmed = errors.New("input source not confirmed")

func waitForInputSource(
	ctx context.Context,
	target, previous uint32,
	attempts int,
	delay time.Duration,
	read func(context.Context) (uint32, error),
) error {
	sawValue := false
	for range attempts {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}

		current, err := read(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil || current == 0 {
			continue
		}
		sawValue = true
		if current == target {
			return nil
		}
	}
	if !sawValue && previous > 0 {
		return nil
	}
	return errInputSourceNotConfirmed
}

func toggleTarget(current uint32, known bool, portA, portB uint32, state int) (target, previous uint32, nextState int) {
	if known && current > 0 {
		previous = current
		if current == portA {
			return portB, previous, 1
		}
		if current == portB {
			return portA, previous, 0
		}
	}
	if state == 0 {
		return portB, previous, 1
	}
	return portA, previous, 0
}

type InputSwitch struct {
	mgr  display.Manager
	resp Responder
}

func NewInputSwitch(mgr display.Manager, resp Responder) *InputSwitch {
	return &InputSwitch{mgr: mgr, resp: resp}
}

type inputSettings struct {
	Mode         string `json:"mode"`
	Port         uint32 `json:"port"`
	PortA        uint32 `json:"portA"`
	PortB        uint32 `json:"portB"`
	MonitorID    string `json:"monitorId"`
	CurrentState int    `json:"currentState"` // 0 = Port A, 1 = Port B
}

func (a *InputSwitch) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	var s inputSettings
	_ = parseSettings(ev, &s)
	if s.MonitorID == "" {
		_ = a.resp.ShowAlert(ev.Context)
		return display.ErrNoMonitorsSelected
	}

	if s.Port == 0 {
		s.Port = 15 // DisplayPort 1
	}
	if s.PortA == 0 {
		s.PortA = 15 // DisplayPort 1
	}
	if s.PortB == 0 {
		s.PortB = 17 // HDMI 1
	}

	var target uint32
	var previous uint32
	var nextState int
	if s.Mode == "toggle" {
		cur, err := a.mgr.GetInputSource(ctx, s.MonitorID)
		target, previous, nextState = toggleTarget(cur, err == nil, s.PortA, s.PortB, s.CurrentState)
	} else {
		target = s.Port
		_ = a.resp.SetState(ev.Context, 0)
	}

	if err := a.mgr.SetInputSource(ctx, s.MonitorID, target); err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}

	if s.Mode == "toggle" {
		err := waitForInputSource(ctx, target, previous, inputConfirmationAttempts, inputConfirmationDelay, func(ctx context.Context) (uint32, error) {
			return a.mgr.GetInputSource(ctx, s.MonitorID)
		})
		if err != nil {
			_ = a.resp.ShowAlert(ev.Context)
			return err
		}

		s.CurrentState = nextState
		_ = a.resp.SetSettings(ev.Context, s)
		_ = a.resp.SetState(ev.Context, nextState)
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

	_ = a.resp.SetState(ev.Context, s.CurrentState)
	return nil
}

func (a *InputSwitch) OnPropertyInspectorDidAppear(ctx context.Context, ev streamdeck.Event) error {
	return sendMonitorsPayload(ctx, a.mgr, a.resp, ev)
}

func (a *InputSwitch) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	_ = HandleCommonPluginMessage(ctx, a.mgr, a.resp, ev)
	return nil
}
