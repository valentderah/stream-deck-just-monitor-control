package actions

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

func TestWaitForInputSourceStopsAfterMatch(t *testing.T) {
	reads := 0
	read := func(context.Context) (uint32, error) {
		reads++
		if reads == 1 {
			return 15, nil
		}
		return 17, nil
	}

	err := waitForInputSource(context.Background(), 17, 15, 3, 0, read)
	if err != nil {
		t.Fatalf("waitForInputSource returned error: %v", err)
	}
	if reads != 2 {
		t.Fatalf("read called %d times, want 2", reads)
	}
}

func TestWaitForInputSourceReturnsNotConfirmedAfterAttempts(t *testing.T) {
	reads := 0
	readErr := errors.New("read failed")
	read := func(context.Context) (uint32, error) {
		reads++
		if reads == 2 {
			return 0, readErr
		}
		return 15, nil
	}

	err := waitForInputSource(context.Background(), 17, 15, 3, 0, read)
	if !errors.Is(err, errInputSourceNotConfirmed) {
		t.Fatalf("waitForInputSource returned %v, want %v", err, errInputSourceNotConfirmed)
	}
	if reads != 3 {
		t.Fatalf("read called %d times, want 3", reads)
	}
}

func TestWaitForInputSourceAcceptsUnreadableMonitorAfterSwitch(t *testing.T) {
	read := func(context.Context) (uint32, error) {
		return 0, errors.New("read failed")
	}

	err := waitForInputSource(context.Background(), 18, 15, 3, 0, read)
	if err != nil {
		t.Fatalf("waitForInputSource returned error: %v", err)
	}
}

func TestWaitForInputSourceRejectsMonitorThatStaysOnPreviousInput(t *testing.T) {
	read := func(context.Context) (uint32, error) {
		return 15, nil
	}

	err := waitForInputSource(context.Background(), 18, 15, 3, 0, read)
	if !errors.Is(err, errInputSourceNotConfirmed) {
		t.Fatalf("waitForInputSource returned %v, want %v", err, errInputSourceNotConfirmed)
	}
}

func settingsEvent(settings string) streamdeck.Event {
	return streamdeck.Event{Context: "ctx", Payload: json.RawMessage(`{"settings":` + settings + `}`)}
}

func TestInputSchemaMatchesSettings(t *testing.T) {
	if err := inspector.Verify(inputSchema, defaultInputSettings()); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeInputSettings(t *testing.T) {
	cases := []struct {
		name     string
		settings string
		want     inputSettings
	}{
		{"missing keys keep defaults", `{"monitorId":"m"}`,
			inputSettings{Mode: InputModeDirect, Port: 15, PortA: 15, PortB: 17, MonitorID: "m"}},
		{"zero ports and empty mode become defaults", `{"mode":"","port":0,"portA":0,"portB":0}`,
			inputSettings{Mode: InputModeDirect, Port: 15, PortA: 15, PortB: 17}},
		{"saved values and current state kept", `{"mode":"toggle","portA":18,"currentState":1}`,
			inputSettings{Mode: InputModeToggle, Port: 15, PortA: 18, PortB: 17, CurrentState: 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := decodeSettings(settingsEvent(c.settings), inputSchema, defaultInputSettings())
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("got %+v want %+v", got, c.want)
			}
		})
	}
}

func TestInputSwitchRejectsInvalidMode(t *testing.T) {
	resp := &fakeResponder{}
	err := NewInputSwitch(&fakeManager{}, resp).OnKeyUp(context.Background(), settingsEvent(`{"monitorId":"m","mode":"bogus"}`))
	if !errors.Is(err, errInvalidMode) {
		t.Fatalf("got %v, want errInvalidMode", err)
	}
	if resp.alerts != 1 {
		t.Fatalf("alerts = %d, want 1", resp.alerts)
	}
}
