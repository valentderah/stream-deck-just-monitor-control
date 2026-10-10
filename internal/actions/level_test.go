package actions

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
)

func TestToRawAndToPercent(t *testing.T) {
	raw := []struct{ percent, max, want uint32 }{
		{50, 100, 50},
		{20, 255, 51},
		{50, 255, 128},
		{100, 255, 255},
		{150, 255, 255},
		{25, 30, 8},
		{0, 30, 0},
	}
	for _, c := range raw {
		if got := toRaw(c.percent, c.max); got != c.want {
			t.Errorf("toRaw(%d, %d) = %d, want %d", c.percent, c.max, got, c.want)
		}
	}
	percent := []struct{ raw, max, want uint32 }{
		{50, 100, 50},
		{51, 255, 20},
		{128, 255, 50},
		{300, 255, 100},
		{8, 30, 27},
		{5, 0, 0},
	}
	for _, c := range percent {
		if got := toPercent(c.raw, c.max); got != c.want {
			t.Errorf("toPercent(%d, %d) = %d, want %d", c.raw, c.max, got, c.want)
		}
	}
}

func TestStepRaw(t *testing.T) {
	cases := []struct {
		current uint32
		step    int32
		max     uint32
		want    uint32
	}{
		{50, 10, 100, 60},
		{95, 10, 100, 100},
		{5, -10, 100, 0},
		{128, 5, 255, 141},
		{0, -5, 255, 0},
		{10, 1, 30, 11},
		{10, -1, 30, 9},
		{30, 1, 30, 30},
		{10, 0, 30, 10},
	}
	for _, c := range cases {
		if got := stepRaw(c.current, c.step, c.max); got != c.want {
			t.Errorf("stepRaw(%d, %d, %d) = %d, want %d", c.current, c.step, c.max, got, c.want)
		}
	}
}

func TestLevelTarget(t *testing.T) {
	s := levelSettings{Value: 50, Step: 10, ToggleA: 25, ToggleB: 75}
	cases := []struct {
		mode         LevelMode
		current, max uint32
		want         uint32
	}{
		{LevelModeSet, 10, 100, 50},
		{LevelModeSet, 10, 255, 128},
		{LevelModeStep, 95, 100, 100},
		{LevelModeToggle, 25, 100, 75},
		{LevelModeToggle, 75, 100, 25},
		{LevelModeToggle, 40, 100, 25},
		{LevelModeToggle, 64, 255, 191},
		{LevelModeToggle, 191, 255, 64},
		// 25% of 30 is raw 8, which reads back as 27%. Comparing raw values keeps the toggle working.
		{LevelModeToggle, 8, 30, 23},
		{LevelModeToggle, 23, 30, 8},
	}
	for _, c := range cases {
		s.Mode = c.mode
		got, err := levelTarget(s, c.current, c.max)
		if err != nil || got != c.want {
			t.Errorf("%s from %d/%d: got %d, %v; want %d", c.mode, c.current, c.max, got, err, c.want)
		}
	}
	for _, mode := range []LevelMode{LevelModeMute, "bogus", ""} {
		s.Mode = mode
		if _, err := levelTarget(s, 0, 100); !errors.Is(err, errInvalidMode) {
			t.Errorf("mode %q: got %v, want errInvalidMode", mode, err)
		}
	}
}

const testCode byte = 0x12

func testLevel(mgr *fakeManager, resp *fakeResponder) *Level {
	defaults := levelSettings{Mode: LevelModeSet, Value: 50, Step: 5, ToggleA: 25, ToggleB: 75}
	dialDefaults := defaults
	dialDefaults.Mode = LevelModeToggle
	return newLevel(mgr, resp, vcpIO{mgr, testCode}, keyProfile(defaults, levelModes), dialProfile(dialDefaults, levelDialModes))
}

func TestLevelWritesRawPerMonitor(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{
		{"a", testCode}: {10, 100},
		{"b", testCode}: {10, 255},
	}}
	resp := &fakeResponder{}
	err := testLevel(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["a","b"],"mode":"set","value":50}`))
	if err != nil {
		t.Fatal(err)
	}
	if a, b := mgr.level("a", testCode), mgr.level("b", testCode); a != 50 || b != 128 {
		t.Fatalf("wrote a=%d b=%d, want 50 and 128", a, b)
	}
	if resp.oks != 1 || resp.alerts != 0 || len(resp.titles) != 0 {
		t.Fatalf("oks=%d alerts=%d titles=%v, want 1, 0, none", resp.oks, resp.alerts, resp.titles)
	}
}

func TestLevelSingleMonitorShowsPercentTitle(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", testCode}: {128, 255}}}
	resp := &fakeResponder{}
	err := testLevel(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"step","step":5}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := mgr.level("m", testCode); got != 141 {
		t.Fatalf("wrote %d, want 141", got)
	}
	if !reflect.DeepEqual(resp.titles, []string{"55%"}) {
		t.Fatalf("titles = %v, want [55%%]", resp.titles)
	}
}

func TestLevelToggleOnShortRange(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", testCode}: {8, 30}}}
	l := testLevel(mgr, &fakeResponder{})
	ev := settingsEvent(`{"monitorIds":["m"],"mode":"toggle","toggleA":25,"toggleB":75}`)
	for _, want := range []uint32{23, 8, 23} {
		if err := l.OnKeyUp(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
		if got := mgr.level("m", testCode); got != want {
			t.Fatalf("wrote %d, want %d", got, want)
		}
	}
}

func TestLevelReadErrorAlertsWithoutWriting(t *testing.T) {
	for _, mode := range []string{"set", "step", "toggle"} {
		mgr := &fakeManager{
			levels:  map[fakeKey]fakeLevel{{"m", testCode}: {10, 100}},
			readErr: map[string]error{"m": errBoom},
		}
		resp := &fakeResponder{}
		err := testLevel(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"`+mode+`"}`))
		if !errors.Is(err, errBoom) || resp.alerts != 1 || resp.oks != 0 {
			t.Errorf("%s: err=%v alerts=%d oks=%d, want errBoom, 1, 0", mode, err, resp.alerts, resp.oks)
		}
		if got := mgr.level("m", testCode); got != 10 {
			t.Errorf("%s: level changed to %d", mode, got)
		}
	}
}

func TestLevelZeroMaxAlerts(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", testCode}: {5, 0}}}
	resp := &fakeResponder{}
	err := testLevel(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"set","value":50}`))
	if !errors.Is(err, errNoLevelRange) || resp.alerts != 1 || resp.oks != 0 {
		t.Fatalf("err=%v alerts=%d oks=%d, want errNoLevelRange, 1, 0", err, resp.alerts, resp.oks)
	}
	if got := mgr.level("m", testCode); got != 5 {
		t.Fatalf("level is %d, want 5 (no write)", got)
	}
}

func TestLevelClampsCurrentAboveMax(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", testCode}: {300, 255}}}
	err := testLevel(mgr, &fakeResponder{}).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"step","step":-5}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := mgr.level("m", testCode); got != 242 {
		t.Fatalf("wrote %d, want 242", got)
	}
}

func TestLevelOneMonitorFailsOthersWritten(t *testing.T) {
	mgr := &fakeManager{
		levels: map[fakeKey]fakeLevel{
			{"a", testCode}: {10, 100},
			{"b", testCode}: {10, 100},
		},
		writeErr: map[string]error{"b": errBoom},
	}
	resp := &fakeResponder{}
	err := testLevel(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["a","b"],"mode":"set","value":50}`))
	if !errors.Is(err, errBoom) || resp.alerts != 1 || resp.oks != 0 {
		t.Fatalf("err=%v alerts=%d oks=%d, want errBoom, 1, 0", err, resp.alerts, resp.oks)
	}
	if a, b := mgr.level("a", testCode), mgr.level("b", testCode); a != 50 || b != 10 {
		t.Fatalf("a=%d b=%d, want 50 and 10", a, b)
	}
}

func TestLevelRejectsBadSettings(t *testing.T) {
	cases := []struct {
		settings string
		want     error
	}{
		{`{"mode":"set"}`, display.ErrNoMonitorsSelected},
		{`{"monitorIds":["m"],"mode":"bogus"}`, errInvalidMode},
		{`{"monitorIds":["m"],"mode":"mute"}`, errInvalidMode},
	}
	for _, c := range cases {
		resp := &fakeResponder{}
		err := testLevel(&fakeManager{}, resp).OnKeyUp(context.Background(), settingsEvent(c.settings))
		if !errors.Is(err, c.want) || resp.alerts != 1 {
			t.Errorf("%s: err=%v alerts=%d, want %v and 1", c.settings, err, resp.alerts, c.want)
		}
	}
}
