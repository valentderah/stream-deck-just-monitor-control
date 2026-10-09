package actions

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
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

func TestPropertyInspectorDidAppearSendsNothing(t *testing.T) {
	resp := &fakeResponder{}
	mgr := &fakeManager{}
	handlers := []streamdeck.ActionHandler{
		NewBrightness(mgr, resp),
		NewContrast(mgr, resp),
		NewVolume(mgr, resp),
		NewInputSwitch(mgr, resp),
		NewRefreshRate(mgr, resp),
		NewHDR(mgr, resp),
		NewSleep(mgr, resp),
		NewRawVCP(mgr, resp),
	}
	for _, h := range handlers {
		if err := h.OnPropertyInspectorDidAppear(context.Background(), pluginMessage(`{}`)); err != nil {
			t.Fatal(err)
		}
	}
	if len(resp.sent) != 0 {
		t.Fatalf("sent %d messages, want 0", len(resp.sent))
	}
}
