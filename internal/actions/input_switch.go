package actions

import (
	"context"
	"errors"
	"time"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
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

type InputMode string

const (
	InputModeDirect InputMode = "direct"
	InputModeToggle InputMode = "toggle"
)

type inputSettings struct {
	Mode         InputMode `json:"mode"`
	Port         uint32    `json:"port"`
	PortA        uint32    `json:"portA"`
	PortB        uint32    `json:"portB"`
	MonitorID    string    `json:"monitorId"`
	CurrentState int       `json:"currentState"` // 0 = Port A, 1 = Port B
}

func defaultInputSettings() inputSettings {
	return inputSettings{Mode: InputModeDirect, Port: 15, PortA: 15, PortB: 17}
}

func inputPortOptions() []inspector.Option {
	ports := display.KnownInputPorts()
	options := make([]inspector.Option, 0, len(ports))
	for _, p := range ports {
		options = append(options, inspector.LiteralOption(p.Code, p.Name))
	}
	return options
}

var inputSchema = inspector.MustBuild(defaultInputSettings(),
	inspector.MonitorField(),
	inspector.Select("mode", "Mode", inspector.TypeString,
		inspector.LocalizedOption(InputModeDirect, "Direct"),
		inspector.LocalizedOption(InputModeToggle, "Toggle"),
	).WithZeroAsUnset(),
	inspector.SourceSelect("port", "Port", inspector.SourceMonitorInputs, inputPortOptions()).
		WithZeroAsUnset().
		VisibleIf("mode", InputModeDirect),
	inspector.SourceSelect("portA", "PortA", inspector.SourceMonitorInputs, inputPortOptions()).
		WithZeroAsUnset().
		VisibleIf("mode", InputModeToggle),
	inspector.SourceSelect("portB", "PortB", inspector.SourceMonitorInputs, inputPortOptions()).
		WithZeroAsUnset().
		VisibleIf("mode", InputModeToggle),
)

func (a *InputSwitch) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	s, err := decodeSettings(ev, inputSchema, defaultInputSettings())
	if err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}
	if s.MonitorID == "" {
		_ = a.resp.ShowAlert(ev.Context)
		return display.ErrNoMonitorsSelected
	}

	var target, previous uint32
	var nextState int
	switch s.Mode {
	case InputModeToggle:
		cur, err := a.mgr.GetInputSource(ctx, s.MonitorID)
		target, previous, nextState = toggleTarget(cur, err == nil, s.PortA, s.PortB, s.CurrentState)
	case InputModeDirect:
		target = s.Port
		_ = a.resp.SetState(ev.Context, 0)
	default:
		_ = a.resp.ShowAlert(ev.Context)
		return errInvalidMode
	}

	if err := a.mgr.SetInputSource(ctx, s.MonitorID, target); err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
	}

	if s.Mode == InputModeToggle {
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
	s, _ := decodeSettings(ev, inputSchema, defaultInputSettings())
	if s.MonitorID == "" || s.Mode != InputModeToggle {
		return nil
	}
	_ = a.resp.SetState(ev.Context, s.CurrentState)
	return nil
}

func (a *InputSwitch) OnPropertyInspectorDidAppear(context.Context, streamdeck.Event) error {
	return nil
}

func (a *InputSwitch) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	_ = HandleCommonPluginMessage(ctx, a.mgr, a.resp, ev, inputSchema)
	return nil
}
