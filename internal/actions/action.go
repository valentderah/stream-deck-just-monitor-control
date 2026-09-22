package actions

import "github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"

type Responder interface {
	SetTitle(context, title string) error
	ShowAlert(context string) error
	ShowOk(context string) error
	SetState(context string, state int) error
	SendToPropertyInspector(context, action string, payload any) error
	SetSettings(context string, settings any) error
}

// Ensure Client satisfies Responder at compile time when used.
var _ Responder = (*streamdeck.Client)(nil)
