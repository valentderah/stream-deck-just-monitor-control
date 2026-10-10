package actions

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
)

const muteSettings = `{"monitorIds":["a","b"],"mode":"mute"}`

func muteManager(a, b uint32) *fakeManager {
	return &fakeManager{levels: map[fakeKey]fakeLevel{
		{"a", 0x8D}: {a, 2},
		{"b", 0x8D}: {b, 2},
	}}
}

func TestVolumeSchemaMatchesSettings(t *testing.T) {
	if err := inspector.Verify(volumeKey.schema, defaultVolumeSettings()); err != nil {
		t.Fatal(err)
	}
}

func TestVolumeDefaults(t *testing.T) {
	got, err := decodeSettings(settingsEvent(`{}`), volumeKey.schema, defaultVolumeSettings())
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != LevelModeSet || got.Value != 50 || got.Step != 5 || got.ToggleA != 20 || got.ToggleB != 60 {
		t.Fatalf("got %+v", got)
	}
}

func TestVolumeSetWritesVCP62(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", 0x62}: {10, 100}}}
	resp := &fakeResponder{}
	if err := NewVolume(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"set","value":30}`)); err != nil {
		t.Fatal(err)
	}
	if got := mgr.level("m", 0x62); got != 30 {
		t.Fatalf("wrote %d, want 30", got)
	}
	if !reflect.DeepEqual(resp.titles, []string{"30%"}) || !reflect.DeepEqual(resp.states, []int{0}) {
		t.Fatalf("titles=%v states=%v, want [30%%] and [0]", resp.titles, resp.states)
	}
}

func TestVolumeNonMuteKeyPressResetsState(t *testing.T) {
	mgr := &fakeManager{levels: map[fakeKey]fakeLevel{{"m", 0x62}: {10, 100}}}
	resp := &fakeResponder{}
	if err := NewVolume(mgr, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"set","value":30}`)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resp.states, []int{0}) {
		t.Fatalf("states=%v, want [0]", resp.states)
	}
}

func TestVolumeMuteToggles(t *testing.T) {
	cases := []struct {
		name      string
		first     uint32
		wantWrite uint32
		wantState int
	}{
		{"unmuted becomes muted", 2, 1, 1},
		{"muted becomes unmuted", 1, 2, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mgr := muteManager(c.first, 2)
			resp := &fakeResponder{}
			if err := NewVolume(mgr, resp).OnKeyUp(context.Background(), settingsEvent(muteSettings)); err != nil {
				t.Fatal(err)
			}
			if a, b := mgr.level("a", 0x8D), mgr.level("b", 0x8D); a != c.wantWrite || b != c.wantWrite {
				t.Fatalf("wrote a=%d b=%d, want %d on both", a, b, c.wantWrite)
			}
			if !reflect.DeepEqual(resp.states, []int{c.wantState}) || resp.oks != 1 || len(resp.titles) != 0 {
				t.Fatalf("states=%v oks=%d titles=%v", resp.states, resp.oks, resp.titles)
			}
		})
	}
}

func TestVolumeMuteUnknownValueMutes(t *testing.T) {
	mgr := muteManager(0, 0)
	resp := &fakeResponder{}
	if err := NewVolume(mgr, resp).OnKeyUp(context.Background(), settingsEvent(muteSettings)); err != nil {
		t.Fatal(err)
	}
	if got := mgr.level("a", 0x8D); got != 1 || !reflect.DeepEqual(resp.states, []int{1}) {
		t.Fatalf("wrote %d with states %v, want 1 and [1]", got, resp.states)
	}
}

func TestVolumeMuteReadErrorAlerts(t *testing.T) {
	mgr := muteManager(2, 2)
	mgr.readErr = map[string]error{"a": errBoom}
	resp := &fakeResponder{}
	err := NewVolume(mgr, resp).OnKeyUp(context.Background(), settingsEvent(muteSettings))
	if !errors.Is(err, errBoom) || resp.alerts != 1 || resp.oks != 0 || len(resp.states) != 0 {
		t.Fatalf("err=%v alerts=%d oks=%d states=%v", err, resp.alerts, resp.oks, resp.states)
	}
	if got := mgr.level("b", 0x8D); got != 2 {
		t.Fatalf("wrote %d to b, want no write", got)
	}
}

func TestVolumeMuteWriteErrorRestoresState(t *testing.T) {
	mgr := muteManager(1, 1)
	mgr.writeErr = map[string]error{"b": errBoom}
	resp := &fakeResponder{}
	err := NewVolume(mgr, resp).OnKeyUp(context.Background(), settingsEvent(muteSettings))
	if !errors.Is(err, errBoom) || resp.alerts != 1 || resp.oks != 0 {
		t.Fatalf("err=%v alerts=%d oks=%d", err, resp.alerts, resp.oks)
	}
	if !reflect.DeepEqual(resp.states, []int{1}) {
		t.Fatalf("states=%v, want [1] (still muted)", resp.states)
	}
}

func TestVolumeWillAppearSetsState(t *testing.T) {
	cases := []struct {
		name     string
		settings string
		mgr      *fakeManager
		want     int
	}{
		{"mute mode, muted", muteSettings, muteManager(1, 2), 1},
		{"mute mode, unmuted", muteSettings, muteManager(2, 1), 0},
		{"mute mode, read fails", muteSettings, &fakeManager{}, 0},
		{"mute mode, no monitors", `{"mode":"mute"}`, muteManager(1, 1), 0},
		{"set mode ignores mute", `{"monitorIds":["a"],"mode":"set"}`, muteManager(1, 1), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := &fakeResponder{}
			if err := NewVolume(c.mgr, resp).OnWillAppear(context.Background(), settingsEvent(c.settings)); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(resp.states, []int{c.want}) {
				t.Fatalf("states=%v, want [%d]", resp.states, c.want)
			}
		})
	}
}

func TestVolumeRejectsBogusMode(t *testing.T) {
	resp := &fakeResponder{}
	err := NewVolume(&fakeManager{}, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorIds":["m"],"mode":"bogus"}`))
	if !errors.Is(err, errInvalidMode) || resp.alerts != 1 {
		t.Fatalf("got %v with %d alerts, want errInvalidMode and 1", err, resp.alerts)
	}
}
