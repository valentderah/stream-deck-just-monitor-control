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

// inspectorRequest is a message the property inspector sends to the plugin.
type inspectorRequest struct {
	Type       string `json:"type"`
	Controller string `json:"controller"`
	MonitorID  string `json:"monitorId"`
}

// parsePluginMessage returns the zero request for a payload it cannot read.
func parsePluginMessage(ev streamdeck.Event) inspectorRequest {
	var m inspectorRequest
	if len(ev.Payload) > 0 {
		_ = json.Unmarshal(ev.Payload, &m)
	}
	return m
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

// handleInspectorMessage serves the messages every property inspector sends and reports whether msg was one of them.
func handleInspectorMessage(ctx context.Context, mgr display.Manager, resp Responder, ev streamdeck.Event, msg inspectorRequest, schema inspector.Schema) (bool, error) {
	switch msg.Type {
	case "get_inspector":
		err := resp.SendToPropertyInspector(ev.Context, ev.Action, map[string]any{
			"type":   "schema",
			"schema": schema,
		})
		return true, errors.Join(err, sendMonitorsPayload(ctx, mgr, resp, ev))
	case "get_monitors":
		return true, sendMonitorsPayload(ctx, mgr, resp, ev)
	case "identify":
		return true, mgr.Identify(ctx)
	}
	return false, nil
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
