package streamdeck

import "context"

type ActionHandler interface {
	OnKeyUp(ctx context.Context, ev Event) error
	OnWillAppear(ctx context.Context, ev Event) error
	OnPropertyInspectorDidAppear(ctx context.Context, ev Event) error
	OnSendToPlugin(ctx context.Context, ev Event) error
}

type Router struct {
	handlers map[string]ActionHandler
}

func NewRouter() *Router {
	return &Router{handlers: make(map[string]ActionHandler)}
}

func (r *Router) Register(uuid string, h ActionHandler) {
	r.handlers[uuid] = h
}

func (r *Router) Handle(ev Event) {
	h, ok := r.handlers[ev.Action]
	if !ok {
		return
	}
	switch ev.Event {
	case "keyUp":
		go func() { _ = h.OnKeyUp(context.Background(), ev) }()
	case "willAppear":
		go func() { _ = h.OnWillAppear(context.Background(), ev) }()
	case "propertyInspectorDidAppear":
		go func() { _ = h.OnPropertyInspectorDidAppear(context.Background(), ev) }()
	case "sendToPlugin":
		go func() { _ = h.OnSendToPlugin(context.Background(), ev) }()
	default:
		// ignore other events for v1
	}
}
