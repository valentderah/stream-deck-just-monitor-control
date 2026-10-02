package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
)

type fakeResponder struct {
	alerts int
	sent   []map[string]any
}

func (r *fakeResponder) SetTitle(string, string) error { return nil }
func (r *fakeResponder) ShowAlert(string) error        { r.alerts++; return nil }
func (r *fakeResponder) ShowOk(string) error           { return nil }
func (r *fakeResponder) SetState(string, int) error    { return nil }
func (r *fakeResponder) SetSettings(string, any) error { return nil }

func (r *fakeResponder) SendToPropertyInspector(_, _ string, payload any) error {
	r.sent = append(r.sent, payload.(map[string]any))
	return nil
}

// fakeManager panics on any Manager method it does not override.
type fakeManager struct {
	display.Manager
	monitors    []display.Monitor
	monitorsErr error
}

func (m *fakeManager) GetMonitors(context.Context) ([]display.Monitor, error) {
	return m.monitors, m.monitorsErr
}
