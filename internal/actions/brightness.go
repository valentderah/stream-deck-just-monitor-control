package actions

import (
	"context"
	"fmt"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

type Brightness struct {
	mgr  display.Manager
	resp Responder
}

func NewBrightness(mgr display.Manager, resp Responder) *Brightness {
	return &Brightness{mgr: mgr, resp: resp}
}

type brightnessSettings struct {
	Mode       string   `json:"mode"`
	Value      uint32   `json:"value"`
	Step       int32    `json:"step"`
	ToggleA    uint32   `json:"toggleA"`
	ToggleB    uint32   `json:"toggleB"`
	MonitorIDs []string `json:"monitorIds"`
}

func applyStep(current uint32, step int32, max uint32) uint32 {
	v := int32(current) + step
	if v < 0 {
		return 0
	}
	if uint32(v) > max {
		return max
	}
	return uint32(v)
}

func applyToggle(current, a, b uint32) uint32 {
	if current == a {
		return b
	}
	return a
}

func (b *Brightness) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	var s brightnessSettings
	_ = parseSettings(ev, &s)
	if s.Mode == "" {
		s.Mode = "set"
	}
	if len(s.MonitorIDs) == 0 {
		_ = b.resp.ShowAlert(ev.Context)
		return display.ErrNoMonitorsSelected
	}

	err := forEachMonitorParallel(s.MonitorIDs, func(id string) error {
		cur, err := b.mgr.GetBrightness(ctx, id)
		if err != nil {
			cur = s.Value
		}
		var next uint32
		switch s.Mode {
		case "step":
			step := s.Step
			if step == 0 {
				step = 10
			}
			next = applyStep(cur, step, 100)
		case "toggle":
			next = applyToggle(cur, s.ToggleA, s.ToggleB)
		default:
			next = s.Value
		}
		return b.mgr.SetBrightness(ctx, id, next)
	})

	if err != nil {
		_ = b.resp.ShowAlert(ev.Context)
		return err
	}

	if len(s.MonitorIDs) == 1 {
		if v, err := b.mgr.GetBrightness(ctx, s.MonitorIDs[0]); err == nil {
			_ = b.resp.SetTitle(ev.Context, fmt.Sprintf("%d%%", v))
		}
	}
	_ = b.resp.ShowOk(ev.Context)
	return nil
}

func (b *Brightness) OnWillAppear(ctx context.Context, ev streamdeck.Event) error {
	return nil
}

func (b *Brightness) OnPropertyInspectorDidAppear(ctx context.Context, ev streamdeck.Event) error {
	return sendMonitorsPayload(ctx, b.mgr, b.resp, ev)
}

func (b *Brightness) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	_ = HandleCommonPluginMessage(ctx, b.mgr, b.resp, ev)
	return nil
}
