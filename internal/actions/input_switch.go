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

	// Дефолтные порты, если пользователь не менял селекторы
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
	var nextState int

	switch s.Mode {
	case "toggle":
		cur, err := a.mgr.GetInputSource(ctx, s.MonitorID)
		if err == nil && cur > 0 {
			if cur == s.PortA {
				// Монитор точно на Port A -> переключаем на Port B
				target = s.PortB
				nextState = 1
			} else if cur == s.PortB {
				// Монитор точно на Port B -> переключаем на Port A
				target = s.PortA
				nextState = 0
			} else {
				// Монитор на стороннем входе -> переключаем по сохраненному состоянию кнопки
				if s.CurrentState == 0 {
					target = s.PortB
					nextState = 1
				} else {
					target = s.PortA
					nextState = 0
				}
			}
		} else {
			// Чтение не удалось или не поддерживается -> надежно шагаем по сохраненному стейту
			if s.CurrentState == 0 {
				target = s.PortB
				nextState = 1
			} else {
				target = s.PortA
				nextState = 0
			}
		}

		s.CurrentState = nextState
		_ = a.resp.SetSettings(ev.Context, s)
		_ = a.resp.SetState(ev.Context, nextState)

	default: // Direct
		target = s.Port
		nextState = 0
		_ = a.resp.SetState(ev.Context, 0)
	}

	if err := a.mgr.SetInputSource(ctx, s.MonitorID, target); err != nil {
		_ = a.resp.ShowAlert(ev.Context)
		return err
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

	if s.PortA == 0 {
		s.PortA = 15
	}
	if s.PortB == 0 {
		s.PortB = 17
	}

	// Синхронизируем состояние при старте, если монитор отвечает
	if cur, err := a.mgr.GetInputSource(ctx, s.MonitorID); err == nil && cur > 0 {
		if cur == s.PortB {
			s.CurrentState = 1
		} else if cur == s.PortA {
			s.CurrentState = 0
		}
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
