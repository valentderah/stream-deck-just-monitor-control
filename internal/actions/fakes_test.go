package actions

import (
	"context"
	"errors"
	"sync"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
)

var errBoom = errors.New("boom")

type fakeResponder struct {
	alerts int
	oks    int
	titles []string
	states []int
	sent   []map[string]any
}

func (r *fakeResponder) SetTitle(_, title string) error { r.titles = append(r.titles, title); return nil }
func (r *fakeResponder) ShowAlert(string) error         { r.alerts++; return nil }
func (r *fakeResponder) ShowOk(string) error            { r.oks++; return nil }
func (r *fakeResponder) SetState(_ string, state int) error {
	r.states = append(r.states, state)
	return nil
}
func (r *fakeResponder) SetSettings(string, any) error { return nil }

func (r *fakeResponder) SendToPropertyInspector(_, _ string, payload any) error {
	r.sent = append(r.sent, payload.(map[string]any))
	return nil
}

const fakeBrightnessCode byte = 0x10

type fakeKey struct {
	id   string
	code byte
}

type fakeLevel struct{ cur, max uint32 }

// fakeManager panics on any Manager method it does not override.
type fakeManager struct {
	display.Manager
	monitors    []display.Monitor
	monitorsErr error

	mu       sync.Mutex
	levels   map[fakeKey]fakeLevel
	readErr  map[string]error
	writeErr map[string]error
}

func (m *fakeManager) GetMonitors(context.Context) ([]display.Monitor, error) {
	return m.monitors, m.monitorsErr
}

func (m *fakeManager) GetVCP(_ context.Context, id string, code byte) (uint32, uint32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.readErr[id]; err != nil {
		return 0, 0, err
	}
	l, ok := m.levels[fakeKey{id, code}]
	if !ok {
		return 0, 0, display.ErrMonitorNotFound
	}
	return l.cur, l.max, nil
}

func (m *fakeManager) SetVCP(_ context.Context, id string, code byte, value uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.writeErr[id]; err != nil {
		return err
	}
	if m.levels == nil {
		m.levels = map[fakeKey]fakeLevel{}
	}
	l := m.levels[fakeKey{id, code}]
	l.cur = value
	m.levels[fakeKey{id, code}] = l
	return nil
}

func (m *fakeManager) GetBrightness(ctx context.Context, id string) (uint32, error) {
	cur, _, err := m.GetVCP(ctx, id, fakeBrightnessCode)
	return cur, err
}

func (m *fakeManager) SetBrightness(ctx context.Context, id string, value uint32) error {
	return m.SetVCP(ctx, id, fakeBrightnessCode, value)
}

func (m *fakeManager) level(id string, code byte) uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.levels[fakeKey{id, code}].cur
}
