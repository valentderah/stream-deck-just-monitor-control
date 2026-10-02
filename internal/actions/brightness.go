package actions

import (
	"context"
	"fmt"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

type Brightness struct {
	mgr  display.Manager
	resp Responder
}

func NewBrightness(mgr display.Manager, resp Responder) *Brightness {
	return &Brightness{mgr: mgr, resp: resp}
}

type BrightnessMode string

const (
	BrightnessModeSet    BrightnessMode = "set"
	BrightnessModeStep   BrightnessMode = "step"
	BrightnessModeToggle BrightnessMode = "toggle"
)

type brightnessSettings struct {
	Mode       BrightnessMode `json:"mode"`
	Value      uint32         `json:"value"`
	Step       int32          `json:"step"`
	ToggleA    uint32         `json:"toggleA"`
	ToggleB    uint32         `json:"toggleB"`
	MonitorIDs []string       `json:"monitorIds"`
}

func defaultBrightnessSettings() brightnessSettings {
	return brightnessSettings{Mode: BrightnessModeSet, Value: 50, Step: 10, ToggleA: 20, ToggleB: 80}
}

var brightnessSchema = inspector.MustBuild(defaultBrightnessSettings(),
	inspector.MonitorsField(),
	inspector.Select("mode", "Mode", inspector.TypeString,
		inspector.LocalizedOption(BrightnessModeSet, "Set"),
		inspector.LocalizedOption(BrightnessModeStep, "Step"),
		inspector.LocalizedOption(BrightnessModeToggle, "Toggle"),
	).WithZeroAsUnset(),
	inspector.Number("value", "Value").WithRange(0, 100).VisibleIf("mode", BrightnessModeSet),
	inspector.Number("step", "StepAmount").WithZeroAsUnset().VisibleIf("mode", BrightnessModeStep),
	inspector.Number("toggleA", "ToggleA").WithRange(0, 100).VisibleIf("mode", BrightnessModeToggle),
	inspector.Number("toggleB", "ToggleB").WithRange(0, 100).VisibleIf("mode", BrightnessModeToggle),
)

func brightnessTarget(s brightnessSettings, current uint32) (uint32, error) {
	switch s.Mode {
	case BrightnessModeSet:
		return s.Value, nil
	case BrightnessModeStep:
		return applyStep(current, s.Step, 100), nil
	case BrightnessModeToggle:
		return applyToggle(current, s.ToggleA, s.ToggleB), nil
	default:
		return 0, errInvalidMode
	}
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
	s, err := decodeSettings(ev, brightnessSchema, defaultBrightnessSettings())
	if err != nil {
		_ = b.resp.ShowAlert(ev.Context)
		return err
	}
	if len(s.MonitorIDs) == 0 {
		_ = b.resp.ShowAlert(ev.Context)
		return display.ErrNoMonitorsSelected
	}
	if _, err := brightnessTarget(s, 0); err != nil {
		_ = b.resp.ShowAlert(ev.Context)
		return err
	}

	err = forEachMonitorParallel(s.MonitorIDs, func(id string) error {
		cur, err := b.mgr.GetBrightness(ctx, id)
		if err != nil {
			cur = s.Value
		}
		next, _ := brightnessTarget(s, cur)
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

func (b *Brightness) OnPropertyInspectorDidAppear(context.Context, streamdeck.Event) error {
	return nil
}

func (b *Brightness) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	_ = HandleCommonPluginMessage(ctx, b.mgr, b.resp, ev, brightnessSchema)
	return nil
}
