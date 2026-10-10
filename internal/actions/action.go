package actions

import (
	"context"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/inspector"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

type Responder interface {
	SetTitle(context, title string) error
	ShowAlert(context string) error
	ShowOk(context string) error
	SetState(context string, state int) error
	SetFeedback(context string, payload any) error
	SendToPropertyInspector(context, action string, payload any) error
	SetSettings(context string, settings any) error
}

// Ensure Client satisfies Responder at compile time when used.
var _ Responder = (*streamdeck.Client)(nil)

const uuidPrefix = "com.valentderah.just-monitor-control."

// All maps every action UUID declared in manifest.json to its handler.
func All(mgr display.Manager, resp Responder) map[string]streamdeck.ActionHandler {
	return map[string]streamdeck.ActionHandler{
		uuidPrefix + "brightness":   NewBrightness(mgr, resp),
		uuidPrefix + "contrast":     NewContrast(mgr, resp),
		uuidPrefix + "volume":       NewVolume(mgr, resp),
		uuidPrefix + "input-switch": NewInputSwitch(mgr, resp),
		uuidPrefix + "refresh-rate": NewRefreshRate(mgr, resp),
		uuidPrefix + "hdr":          NewHDR(mgr, resp),
		uuidPrefix + "sleep":        NewSleep(mgr, resp),
		uuidPrefix + "raw-vcp":      NewRawVCP(mgr, resp),
	}
}

// inspectorHost answers the property inspector for an action with a single schema.
type inspectorHost struct {
	mgr    display.Manager
	resp   Responder
	schema inspector.Schema
}

func (h inspectorHost) OnSendToPlugin(ctx context.Context, ev streamdeck.Event) error {
	_, err := handleInspectorMessage(ctx, h.mgr, h.resp, ev, parsePluginMessage(ev), h.schema)
	return err
}
