package actions

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

func pluginMessage(payload string) streamdeck.Event {
	return streamdeck.Event{Context: "ctx", Action: "action", Payload: json.RawMessage(payload)}
}

func TestGetInspectorSendsSchemaThenMonitors(t *testing.T) {
	resp := &fakeResponder{}
	mgr := &fakeManager{monitors: []display.Monitor{{ID: "m"}}}
	if err := NewInputSwitch(mgr, resp).OnSendToPlugin(context.Background(), pluginMessage(`{"type":"get_inspector"}`)); err != nil {
		t.Fatal(err)
	}
	if len(resp.sent) != 2 {
		t.Fatalf("sent %d messages, want 2", len(resp.sent))
	}
	if resp.sent[0]["type"] != "schema" || !reflect.DeepEqual(resp.sent[0]["schema"], inputSchema) {
		t.Fatalf("first message = %v", resp.sent[0])
	}
	if resp.sent[1]["type"] != "monitors" || !reflect.DeepEqual(resp.sent[1]["monitors"], mgr.monitors) {
		t.Fatalf("second message = %v", resp.sent[1])
	}
}

func TestSendMonitorsPayloadReturnsContextError(t *testing.T) {
	for _, ctxErr := range []error{context.Canceled, context.DeadlineExceeded} {
		resp := &fakeResponder{}
		err := sendMonitorsPayload(context.Background(), &fakeManager{monitorsErr: ctxErr}, resp, pluginMessage(`{}`))
		if !errors.Is(err, ctxErr) {
			t.Errorf("got %v, want %v", err, ctxErr)
		}
		if len(resp.sent) != 0 {
			t.Errorf("sent %d messages, want 0", len(resp.sent))
		}
	}
}

func TestSendMonitorsPayloadSendsEmptyListOnOtherErrors(t *testing.T) {
	resp := &fakeResponder{}
	err := sendMonitorsPayload(context.Background(), &fakeManager{monitorsErr: errors.New("boom")}, resp, pluginMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	mons, ok := resp.sent[0]["monitors"].([]display.Monitor)
	if !ok || mons == nil || len(mons) != 0 {
		t.Fatalf("monitors = %#v, want empty non-nil list", resp.sent[0]["monitors"])
	}
}

func TestLevelInspectorSchemaFollowsController(t *testing.T) {
	cases := []struct {
		message string
		want    inspector.Schema
	}{
		{`{"type":"get_inspector"}`, brightnessKey.schema},
		{`{"type":"get_inspector","controller":"Keypad"}`, brightnessKey.schema},
		{`{"type":"get_inspector","controller":"Encoder"}`, brightnessDial.schema},
	}
	for _, c := range cases {
		resp := &fakeResponder{}
		if err := NewBrightness(&fakeManager{}, resp).OnSendToPlugin(context.Background(), pluginMessage(c.message)); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(resp.sent[0]["schema"], c.want) {
			t.Errorf("%s: sent the wrong schema", c.message)
		}
	}
}

// Every action UUID lives in both manifest.json and All, and only dial-capable actions may declare the Encoder.
func TestActionsMatchManifest(t *testing.T) {
	raw, err := os.ReadFile("../../assets/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Actions []struct {
			UUID        string
			Controllers []string
		}
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	handlers := All(&fakeManager{}, &fakeResponder{})
	if len(manifest.Actions) != len(handlers) {
		t.Errorf("manifest declares %d actions, All registers %d", len(manifest.Actions), len(handlers))
	}
	for _, a := range manifest.Actions {
		h, ok := handlers[a.UUID]
		if !ok {
			t.Errorf("%s is in the manifest but has no handler", a.UUID)
			continue
		}
		_, handlesDial := h.(streamdeck.DialHandler)
		if declaresDial := slices.Contains(a.Controllers, streamdeck.ControllerEncoder); declaresDial != handlesDial {
			t.Errorf("%s: manifest Encoder=%v, handler implements DialHandler=%v", a.UUID, declaresDial, handlesDial)
		}
	}
}
