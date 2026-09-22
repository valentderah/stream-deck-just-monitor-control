package actions

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

func parseSettings(ev streamdeck.Event, dest any) error {
	if len(ev.Payload) == 0 {
		return nil
	}
	var wrap struct {
		Settings json.RawMessage `json:"settings"`
	}
	if err := json.Unmarshal(ev.Payload, &wrap); err == nil && len(wrap.Settings) > 0 {
		return json.Unmarshal(wrap.Settings, dest)
	}
	return json.Unmarshal(ev.Payload, dest)
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
	if err != nil {
		mons = nil
	}
	return resp.SendToPropertyInspector(ev.Context, ev.Action, map[string]any{
		"type":     "monitors",
		"monitors": mons,
	})
}

func HandleCommonPluginMessage(ctx context.Context, mgr display.Manager, resp Responder, ev streamdeck.Event) bool {
	msg, err := parsePluginMessage(ev)
	if err != nil {
		return false
	}
	switch msg["type"] {
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
