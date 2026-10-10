package actions

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

func dialEvent(settings string) streamdeck.Event {
	return streamdeck.Event{Context: "ctx", Payload: json.RawMessage(`{"controller":"Encoder","settings":` + settings + `}`)}
}

func wantFeedback(t *testing.T, resp *fakeResponder, percent uint32) {
	t.Helper()
	if len(resp.feedbacks) != 1 {
		t.Fatalf("sent %d feedback updates, want 1", len(resp.feedbacks))
	}
	if !reflect.DeepEqual(resp.feedbacks[0], levelFeedback(percent)) {
		t.Fatalf("feedback = %v, want %d%%", resp.feedbacks[0], percent)
	}
}

func TestDialSchemasMatchSettings(t *testing.T) {
	profiles := map[string]levelProfile{
		"brightness": brightnessDial,
		"contrast":   contrastDial,
		"volume":     volumeDial,
	}
	for name, p := range profiles {
		if err := inspector.Verify(p.schema, p.defaults); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestDialStep(t *testing.T) {
	cases := []struct {
		step  int32
		ticks int
		want  int32
	}{
		{5, 1, 5},
		{5, -2, -10},
		{-5, 1, 5},
		{-5, -1, -5},
		{10, 50, 100},
		{10, -50, -100},
	}
	for _, c := range cases {
		if got := dialStep(c.step, c.ticks); got != c.want {
			t.Errorf("dialStep(%d, %d) = %d, want %d", c.step, c.ticks, got, c.want)
		}
	}
}

func TestDialRotateStepsByTicks(t *testing.T) {
	cases := []struct {
		settings string
		ticks    int
		current  uint32
		want     uint32
	}{
		{`{"monitorIds":["m"]}`, 1, 50, 55},
		{`{"monitorIds":["m"]}`, -3, 50, 35},
		{`{"monitorIds":["m"],"step":2}`, 4, 50, 58},
		{`{"monitorIds":["m"],"step":-10}`, 1, 50, 60},
		{`{"monitorIds":["m"],"mode":"set","value":10}`, 1, 50, 55},
		{`{"monitorIds":["m"]}`, 40, 50, 100},
		{`{"monitorIds":["m"]}`, -40, 50, 0},
	}
	for _, c := range cases {
		mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", testCode}: {c.current, 100}}}
		resp := &fakeResponder{}
		if err := testLevel(mgr, resp).OnDialRotate(context.Background(), dialEvent(c.settings), c.ticks); err != nil {
			t.Fatalf("%s by %d: %v", c.settings, c.ticks, err)
		}
		if got := mgr.level("m", testCode); got != c.want {
			t.Errorf("%s by %d from %d: wrote %d, want %d", c.settings, c.ticks, c.current, got, c.want)
		}
		wantFeedback(t, resp, c.want)
		if resp.oks != 0 || len(resp.titles) != 0 {
			t.Errorf("%s: oks=%d titles=%v, want none on a dial", c.settings, resp.oks, resp.titles)
		}
	}
}

func TestDialRotateZeroTicksDoesNothing(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", testCode}: {50, 100}}}
	resp := &fakeResponder{}
	if err := testLevel(mgr, resp).OnDialRotate(context.Background(), dialEvent(`{"monitorIds":["m"]}`), 0); err != nil {
		t.Fatal(err)
	}
	if got := mgr.level("m", testCode); got != 50 || len(resp.feedbacks) != 0 {
		t.Fatalf("level=%d feedbacks=%d, want 50 and 0", got, len(resp.feedbacks))
	}
}

func TestDialRotateShowsFirstMonitor(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{
		{"a", testCode}: {20, 100},
		{"b", testCode}: {70, 100},
	}}
	resp := &fakeResponder{}
	if err := testLevel(mgr, resp).OnDialRotate(context.Background(), dialEvent(`{"monitorIds":["a","b"]}`), 1); err != nil {
		t.Fatal(err)
	}
	if a, b := mgr.level("a", testCode), mgr.level("b", testCode); a != 25 || b != 75 {
		t.Fatalf("wrote a=%d b=%d, want 25 and 75", a, b)
	}
	wantFeedback(t, resp, 25)
}

func TestDialRotateAlertsOnError(t *testing.T) {
	cases := []struct {
		name     string
		settings string
		mgr      *fakeManager
		want     error
	}{
		{"read fails", `{"monitorIds":["m"]}`, &fakeManager{readErr: map[string]error{"m": errBoom}}, errBoom},
		{"no monitors", `{}`, &fakeManager{}, nil},
	}
	for _, c := range cases {
		resp := &fakeResponder{}
		err := testLevel(c.mgr, resp).OnDialRotate(context.Background(), dialEvent(c.settings), 1)
		if err == nil || (c.want != nil && !errors.Is(err, c.want)) {
			t.Errorf("%s: err=%v", c.name, err)
		}
		if resp.alerts != 1 || len(resp.feedbacks) != 0 {
			t.Errorf("%s: alerts=%d feedbacks=%d, want 1 and 0", c.name, resp.alerts, len(resp.feedbacks))
		}
	}
}

func TestDialPressAppliesMode(t *testing.T) {
	cases := []struct {
		settings string
		current  uint32
		want     uint32
	}{
		{`{"monitorIds":["m"]}`, 25, 75},
		{`{"monitorIds":["m"]}`, 75, 25},
		{`{"monitorIds":["m"],"mode":"set","value":40}`, 75, 40},
	}
	for _, c := range cases {
		mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", testCode}: {c.current, 100}}}
		resp := &fakeResponder{}
		if err := testLevel(mgr, resp).OnDialPress(context.Background(), dialEvent(c.settings)); err != nil {
			t.Fatalf("%s: %v", c.settings, err)
		}
		if got := mgr.level("m", testCode); got != c.want {
			t.Errorf("%s from %d: wrote %d, want %d", c.settings, c.current, got, c.want)
		}
		wantFeedback(t, resp, c.want)
	}
}

func TestDialPressRejectsStepMode(t *testing.T) {
	resp := &fakeResponder{}
	err := testLevel(&fakeManager{}, resp).OnDialPress(context.Background(), dialEvent(`{"monitorIds":["m"],"mode":"step"}`))
	if !errors.Is(err, errInvalidMode) || resp.alerts != 1 {
		t.Fatalf("got %v with %d alerts, want errInvalidMode and 1", err, resp.alerts)
	}
}

func TestLevelWillAppearShowsCurrentOnDialOnly(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", testCode}: {128, 255}}}

	resp := &fakeResponder{}
	if err := testLevel(mgr, resp).OnWillAppear(context.Background(), dialEvent(`{"monitorIds":["m"]}`)); err != nil {
		t.Fatal(err)
	}
	wantFeedback(t, resp, 50)

	resp = &fakeResponder{}
	if err := testLevel(mgr, resp).OnWillAppear(context.Background(), settingsEvent(`{"monitorIds":["m"]}`)); err != nil {
		t.Fatal(err)
	}
	if len(resp.feedbacks) != 0 {
		t.Fatalf("sent %d feedback updates to a key, want 0", len(resp.feedbacks))
	}
}

func TestVolumeDialPressMutesByDefault(t *testing.T) {
	mgr := muteManager(2, 2)
	resp := &fakeResponder{}
	if err := NewVolume(mgr, resp).OnDialPress(context.Background(), dialEvent(`{"monitorIds":["a","b"]}`)); err != nil {
		t.Fatal(err)
	}
	if a, b := mgr.level("a", 0x8D), mgr.level("b", 0x8D); a != 1 || b != 1 {
		t.Fatalf("wrote a=%d b=%d, want both muted", a, b)
	}
	if !reflect.DeepEqual(resp.states, []int{volumeStateMuted}) {
		t.Fatalf("states=%v, want [1]", resp.states)
	}
}

func TestVolumeDialRotateStepsVolume(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", 0x62}: {50, 100}}}
	resp := &fakeResponder{}
	if err := NewVolume(mgr, resp).OnDialRotate(context.Background(), dialEvent(`{"monitorIds":["m"]}`), 3); err != nil {
		t.Fatal(err)
	}
	if got := mgr.level("m", 0x62); got != 56 {
		t.Fatalf("wrote %d, want 56", got)
	}
	wantFeedback(t, resp, 56)
}

func TestLevelActionsHandleDials(t *testing.T) {
	mgr, resp := &fakeManager{}, &fakeResponder{}
	for name, h := range map[string]any{
		"brightness": NewBrightness(mgr, resp),
		"contrast":   NewContrast(mgr, resp),
		"volume":     NewVolume(mgr, resp),
	} {
		if _, ok := h.(streamdeck.DialHandler); !ok {
			t.Errorf("%s does not implement streamdeck.DialHandler", name)
		}
	}
}
