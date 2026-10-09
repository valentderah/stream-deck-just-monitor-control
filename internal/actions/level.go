package actions

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

type LevelMode string

const (
	LevelModeSet    LevelMode = "set"
	LevelModeStep   LevelMode = "step"
	LevelModeToggle LevelMode = "toggle"
	LevelModeMute   LevelMode = "mute"
)

// levelSettings holds percentages; the monitor's own range is applied on write.
type levelSettings struct {
	Mode       LevelMode `json:"mode"`
	Value      uint32    `json:"value"`
	Step       int32     `json:"step"`
	ToggleA    uint32    `json:"toggleA"`
	ToggleB    uint32    `json:"toggleB"`
	MonitorIDs []string  `json:"monitorIds"`
}

func toRaw(percent, max uint32) uint32 {
	if percent > 100 {
		percent = 100
	}
	return (percent*max + 50) / 100
}

func toPercent(raw, max uint32) uint32 {
	if max == 0 {
		return 0
	}
	if raw > max {
		raw = max
	}
	return (raw*100 + max/2) / max
}

// stepRaw moves by at least one raw unit so a small step is never a no-op on a short range.
func stepRaw(current uint32, step int32, max uint32) uint32 {
	if current > max {
		current = max
	}
	if step == 0 {
		return current
	}
	magnitude := int64(step)
	if magnitude < 0 {
		magnitude = -magnitude
	}
	delta := (magnitude*int64(max) + 50) / 100
	if delta == 0 {
		delta = 1
	}
	if step < 0 {
		delta = -delta
	}
	v := int64(current) + delta
	if v < 0 {
		return 0
	}
	if v > int64(max) {
		return max
	}
	return uint32(v)
}

func levelTarget(s levelSettings, current, max uint32) (uint32, error) {
	switch s.Mode {
	case LevelModeSet:
		return toRaw(s.Value, max), nil
	case LevelModeStep:
		return stepRaw(current, s.Step, max), nil
	case LevelModeToggle:
		a := toRaw(s.ToggleA, max)
		if current == a {
			return toRaw(s.ToggleB, max), nil
		}
		return a, nil
	default:
		return 0, errInvalidMode
	}
}

var errNoLevelRange = errors.New("actions: monitor reports no range for this control")

var levelModes = []LevelMode{LevelModeSet, LevelModeStep, LevelModeToggle}

var levelModeLabels = map[LevelMode]string{
	LevelModeSet:    "Set",
	LevelModeStep:   "Step",
	LevelModeToggle: "Toggle",
	LevelModeMute:   "Mute",
}

func levelSchema(defaults levelSettings, modes []LevelMode) inspector.Schema {
	options := make([]inspector.Option, len(modes))
	for i, m := range modes {
		options[i] = inspector.LocalizedOption(m, levelModeLabels[m])
	}
	return inspector.MustBuild(defaults,
		inspector.MonitorsField(),
		inspector.Select("mode", "Mode", inspector.TypeString, options...).WithZeroAsUnset(),
		inspector.Number("value", "Value").WithRange(0, 100).VisibleIf("mode", LevelModeSet),
		inspector.Number("step", "StepAmount").WithZeroAsUnset().VisibleIf("mode", LevelModeStep),
		inspector.Number("toggleA", "ToggleA").WithRange(0, 100).VisibleIf("mode", LevelModeToggle),
		inspector.Number("toggleB", "ToggleB").WithRange(0, 100).VisibleIf("mode", LevelModeToggle),
	)
}

// levelIO reads and writes one control on one monitor in the monitor's own units.
// read returns a non-zero max alongside an error only when the range is known without the monitor.
type levelIO interface {
	read(ctx context.Context, monitorID string) (current, max uint32, err error)
	write(ctx context.Context, monitorID string, raw uint32) error
}

type vcpIO struct {
	mgr  display.Manager
	code byte
}

func (v vcpIO) read(ctx context.Context, monitorID string) (uint32, uint32, error) {
	current, max, err := v.mgr.GetVCP(ctx, monitorID, v.code)
	if err != nil {
		return 0, 0, err
	}
	return current, max, nil
}

func (v vcpIO) write(ctx context.Context, monitorID string, raw uint32) error {
	return v.mgr.SetVCP(ctx, monitorID, v.code, raw)
}

type Level struct {
	mgr      display.Manager
	resp     Responder
	io       levelIO
	defaults levelSettings
	schema   inspector.Schema
	modes    []LevelMode
}

func newLevel(mgr display.Manager, resp Responder, io levelIO, defaults levelSettings, schema inspector.Schema, modes []LevelMode) *Level {
	return &Level{mgr: mgr, resp: resp, io: io, defaults: defaults, schema: schema, modes: modes}
}

func (l *Level) settings(ev streamdeck.Event) (levelSettings, error) {
	s, err := decodeSettings(ev, l.schema, l.defaults)
	if err != nil {
		return s, err
	}
	if len(s.MonitorIDs) == 0 {
		return s, display.ErrNoMonitorsSelected
	}
	if !slices.Contains(l.modes, s.Mode) {
		return s, errInvalidMode
	}
	return s, nil
}

func (l *Level) apply(ctx context.Context, ev streamdeck.Event, s levelSettings) error {
	single := len(s.MonitorIDs) == 1
	var title string

	err := forEachMonitorParallel(s.MonitorIDs, func(id string) error {
		current, max, err := l.io.read(ctx, id)
		// Set does not need the current value, only the range.
		if err != nil && !(s.Mode == LevelModeSet && max > 0) {
			return err
		}
		if max == 0 {
			return errNoLevelRange
		}
		if current > max {
			current = max
		}
		next, err := levelTarget(s, current, max)
		if err != nil {
			return err
		}
		if err := l.io.write(ctx, id, next); err != nil {
			return err
		}
		if single {
			title = fmt.Sprintf("%d%%", toPercent(next, max))
		}
		return nil
	})
	if err != nil {
		_ = l.resp.ShowAlert(ev.Context)
		return err
	}

	if single {
		_ = l.resp.SetTitle(ev.Context, title)
	}
	_ = l.resp.ShowOk(ev.Context)
	return nil
}

func (l *Level) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	s, err := l.settings(ev)
	if err != nil {
		_ = l.resp.ShowAlert(ev.Context)
		return err
	}
	return l.apply(ctx, ev, s)
}

func (l *Level) OnWillAppear(context.Context, streamdeck.Event) error { return nil }

func (l *Level) OnPropertyInspectorDidAppear(context.Context, streamdeck.Event) error {
	return nil
}

func (l *Level) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	_ = HandleCommonPluginMessage(ctx, l.mgr, l.resp, ev, l.schema)
	return nil
}
