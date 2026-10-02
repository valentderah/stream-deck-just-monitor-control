package actions

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

var errInvalidMode = errors.New("actions: invalid mode")

func settingsPayload(ev streamdeck.Event) json.RawMessage {
	if len(ev.Payload) == 0 {
		return nil
	}
	var wrap struct {
		Settings json.RawMessage `json:"settings"`
	}
	if err := json.Unmarshal(ev.Payload, &wrap); err == nil && len(wrap.Settings) > 0 {
		return wrap.Settings
	}
	return ev.Payload
}

func decodeSettings[T any](ev streamdeck.Event, schema inspector.Schema, defaults T) (T, error) {
	return inspector.Decode(schema, defaults, settingsPayload(ev))
}

func parsePluginMessage(ev streamdeck.Event) (map[string]any, error) {
	var m map[string]any
	if len(ev.Payload) == 0 {
		return m, nil
	}
	err := json.Unmarshal(ev.Payload, &m)
	return m, err
}

func sendMonitorsPayload(ctx context.Context, mgr display.Manager, resp Responder, ev streamdeck.Event) error {
	mons, err := mgr.GetMonitors(ctx)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if err != nil {
		mons = []display.Monitor{}
	}
	return resp.SendToPropertyInspector(ev.Context, ev.Action, map[string]any{
		"type":     "monitors",
		"monitors": mons,
	})
}

func HandleCommonPluginMessage(ctx context.Context, mgr display.Manager, resp Responder, ev streamdeck.Event, schema inspector.Schema) bool {
	msg, err := parsePluginMessage(ev)
	if err != nil {
		return false
	}
	switch msg["type"] {
	case "get_inspector":
		_ = resp.SendToPropertyInspector(ev.Context, ev.Action, map[string]any{
			"type":   "schema",
			"schema": schema,
		})
		_ = sendMonitorsPayload(ctx, mgr, resp, ev)
		return true
	case "get_monitors":
		_ = sendMonitorsPayload(ctx, mgr, resp, ev)
		return true
	case "identify":
		_ = mgr.Identify(ctx)
		return true
	}
	return false
}

func forEachMonitorParallel(ids []string, fn func(id string) error) error {
	if len(ids) == 0 {
		return nil
	}
	var wg sync.WaitGroup
	errCh := make(chan error, len(ids))
	for _, id := range ids {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(id); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	var first error
	for err := range errCh {
		if first == nil {
			first = err
		}
	}
	return first
}

func liveMonitorIDs(wanted []string, available map[string]struct{}) []string {
	var out []string
	for _, id := range wanted {
		if _, ok := available[id]; ok {
			out = append(out, id)
		}
	}
	return out
}
