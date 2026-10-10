package actions

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

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

// dialStep ignores the sign of the configured step, because the direction comes from the rotation.
func dialStep(step int32, ticks int) int32 {
	magnitude := int64(step)
	if magnitude < 0 {
		magnitude = -magnitude
	}
	return int32(max(-100, min(100, magnitude*int64(ticks))))
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

var (
	levelModes     = []LevelMode{LevelModeSet, LevelModeStep, LevelModeToggle}
	levelDialModes = []LevelMode{LevelModeSet, LevelModeToggle}
)

var levelModeLabels = map[LevelMode]string{
	LevelModeSet:    "Set",
	LevelModeStep:   "Step",
	LevelModeToggle: "Toggle",
	LevelModeMute:   "Mute",
}

func levelModeOptions(modes []LevelMode) []inspector.Option {
	options := make([]inspector.Option, len(modes))
	for i, m := range modes {
		options[i] = inspector.LocalizedOption(m, levelModeLabels[m])
	}
	return options
}

func levelSchema(defaults levelSettings, modes []LevelMode) inspector.Schema {
	return inspector.MustBuild(defaults,
		inspector.MonitorsField(),
		inspector.Select("mode", "Mode", inspector.TypeString, levelModeOptions(modes)...).WithZeroAsUnset(),
		inspector.Number("value", "Value").WithRange(0, 100).VisibleIf("mode", LevelModeSet),
		inspector.Number("step", "StepAmount").WithZeroAsUnset().VisibleIf("mode", LevelModeStep),
		inspector.Number("toggleA", "ToggleA").WithRange(0, 100).VisibleIf("mode", LevelModeToggle),
		inspector.Number("toggleB", "ToggleB").WithRange(0, 100).VisibleIf("mode", LevelModeToggle),
	)
}

// levelDialSchema always shows the step, because rotation steps in every mode. The mode only picks what a press does.
func levelDialSchema(defaults levelSettings, modes []LevelMode) inspector.Schema {
	return inspector.MustBuild(defaults,
		inspector.MonitorsField(),
		inspector.Number("step", "StepAmount").WithRange(1, 100).WithZeroAsUnset(),
		inspector.Select("mode", "PressAction", inspector.TypeString, levelModeOptions(modes)...).WithZeroAsUnset(),
		inspector.Number("value", "Value").WithRange(0, 100).VisibleIf("mode", LevelModeSet),
		inspector.Number("toggleA", "ToggleA").WithRange(0, 100).VisibleIf("mode", LevelModeToggle),
		inspector.Number("toggleB", "ToggleB").WithRange(0, 100).VisibleIf("mode", LevelModeToggle),
	)
}

// levelProfile is how the action is configured on one kind of controller.
type levelProfile struct {
	defaults levelSettings
	schema   inspector.Schema
	modes    []LevelMode
}

func keyProfile(defaults levelSettings, modes []LevelMode) levelProfile {
	return levelProfile{defaults: defaults, schema: levelSchema(defaults, modes), modes: modes}
}

func dialProfile(defaults levelSettings, modes []LevelMode) levelProfile {
	return levelProfile{defaults: defaults, schema: levelDialSchema(defaults, modes), modes: modes}
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
	mgr  display.Manager
	resp Responder
	io   levelIO
	key  levelProfile
	dial levelProfile
	// locks keep two buttons on the same monitor from interleaving their read-modify-write.
	locks *display.IDLocks
}

func newLevel(mgr display.Manager, resp Responder, io levelIO, key, dial levelProfile) *Level {
	return &Level{mgr: mgr, resp: resp, io: io, key: key, dial: dial, locks: display.NewIDLocks()}
}

func (l *Level) profile(ev streamdeck.Event) levelProfile {
	if ev.Controller() == streamdeck.ControllerEncoder {
		return l.dial
	}
	return l.key
}

func (l *Level) settings(ev streamdeck.Event, p levelProfile) (levelSettings, error) {
	s, err := decodeSettings(ev, p.schema, p.defaults)
	if err != nil {
		return s, err
	}
	if len(s.MonitorIDs) == 0 {
		return s, display.ErrNoMonitorsSelected
	}
	if !slices.Contains(p.modes, s.Mode) {
		return s, errInvalidMode
	}
	return s, nil
}

// adjust moves every monitor to target and returns the resulting percentage per monitor.
func (l *Level) adjust(ctx context.Context, monitorIDs []string, needCurrent bool, target func(current, max uint32) (uint32, error)) (map[string]uint32, error) {
	var mu sync.Mutex
	percents := make(map[string]uint32, len(monitorIDs))

	err := forEachMonitorParallel(monitorIDs, func(id string) error {
		unlock, err := l.locks.Lock(ctx, id)
		if err != nil {
			return err
		}
		defer unlock()

		current, max, err := l.io.read(ctx, id)
		// Without the current value only an absolute write is possible, and only when the range is known.
		if err != nil && (needCurrent || max == 0) {
			return err
		}
		if max == 0 {
			return errNoLevelRange
		}
		if current > max {
			current = max
		}
		next, err := target(current, max)
		if err != nil {
			return err
		}
		if err := l.io.write(ctx, id, next); err != nil {
			return err
		}
		mu.Lock()
		percents[id] = toPercent(next, max)
		mu.Unlock()
		return nil
	})
	return percents, err
}

func levelFeedback(percent uint32) map[string]any {
	return map[string]any{
		"value":     fmt.Sprintf("%d%%", percent),
		"indicator": map[string]any{"value": percent},
	}
}

// report shows the first monitor on a dial, because the touch strip has room for one value.
func (l *Level) report(ev streamdeck.Event, onDial bool, monitorIDs []string, percents map[string]uint32) {
	if onDial {
		_ = l.resp.SetFeedback(ev.Context, levelFeedback(percents[monitorIDs[0]]))
		return
	}
	if len(monitorIDs) == 1 {
		_ = l.resp.SetTitle(ev.Context, fmt.Sprintf("%d%%", percents[monitorIDs[0]]))
	}
	_ = l.resp.ShowOk(ev.Context)
}

// press applies the configured mode. A key press and a dial press share it.
func (l *Level) press(ctx context.Context, ev streamdeck.Event, onDial bool, s levelSettings) error {
	percents, err := l.adjust(ctx, s.MonitorIDs, s.Mode != LevelModeSet, func(current, max uint32) (uint32, error) {
		return levelTarget(s, current, max)
	})
	if err != nil {
		_ = l.resp.ShowAlert(ev.Context)
		return err
	}
	l.report(ev, onDial, s.MonitorIDs, percents)
	return nil
}

// showCurrent fills the touch strip when the action appears on a dial.
func (l *Level) showCurrent(ctx context.Context, ev streamdeck.Event) error {
	s, _ := decodeSettings(ev, l.dial.schema, l.dial.defaults)
	if len(s.MonitorIDs) == 0 {
		return nil
	}
	current, max, err := l.io.read(ctx, s.MonitorIDs[0])
	if err != nil {
		return err
	}
	return l.resp.SetFeedback(ev.Context, levelFeedback(toPercent(current, max)))
}

func (l *Level) OnKeyUp(ctx context.Context, ev streamdeck.Event) error {
	s, err := l.settings(ev, l.key)
	if err != nil {
		_ = l.resp.ShowAlert(ev.Context)
		return err
	}
	return l.press(ctx, ev, false, s)
}

func (l *Level) OnDialRotate(ctx context.Context, ev streamdeck.Event, ticks int) error {
	if ticks == 0 {
		return nil
	}
	s, err := l.settings(ev, l.dial)
	if err != nil {
		_ = l.resp.ShowAlert(ev.Context)
		return err
	}
	step := dialStep(s.Step, ticks)
	percents, err := l.adjust(ctx, s.MonitorIDs, true, func(current, max uint32) (uint32, error) {
		return stepRaw(current, step, max), nil
	})
	if err != nil {
		_ = l.resp.ShowAlert(ev.Context)
		return err
	}
	l.report(ev, true, s.MonitorIDs, percents)
	return nil
}

func (l *Level) OnDialPress(ctx context.Context, ev streamdeck.Event) error {
	s, err := l.settings(ev, l.dial)
	if err != nil {
		_ = l.resp.ShowAlert(ev.Context)
		return err
	}
	return l.press(ctx, ev, true, s)
}

func (l *Level) OnWillAppear(ctx context.Context, ev streamdeck.Event) error {
	if ev.Controller() != streamdeck.ControllerEncoder {
		return nil
	}
	return l.showCurrent(ctx, ev)
}

func (l *Level) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	msg := parsePluginMessage(ev)
	schema := l.key.schema
	if msg.Controller == streamdeck.ControllerEncoder {
		schema = l.dial.schema
	}
	_, err := handleInspectorMessage(ctx, l.mgr, l.resp, ev, msg, schema)
	return err
}
