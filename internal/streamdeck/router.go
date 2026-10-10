package streamdeck

import (
	"context"
	"log"
	"sync"
	"time"
)

const (
	defaultActionTimeout    = 20 * time.Second
	defaultInspectorTimeout = 30 * time.Second
	defaultAbandonGrace     = 5 * time.Second
)

type ActionHandler interface {
	OnKeyUp(ctx context.Context, ev Event) error
}

type WillAppearHandler interface {
	OnWillAppear(ctx context.Context, ev Event) error
}

type PluginMessageHandler interface {
	OnSendToPlugin(ctx context.Context, ev Event) error
}

// DialHandler is implemented by actions that declare the Encoder controller.
type DialHandler interface {
	OnDialRotate(ctx context.Context, ev Event, ticks int) error
	// OnDialPress handles both a dial push and a tap on the touch strip.
	OnDialPress(ctx context.Context, ev Event) error
}

type job struct {
	ev     Event
	rotate bool
	ticks  int
	run    func(ctx context.Context, j job) error
}

// Router runs key and dial events of one action instance in order, so a read-modify-write
// on the monitor never interleaves with the next press. Other events run right away.
type Router struct {
	handlers map[string]ActionHandler

	actionTimeout    time.Duration
	inspectorTimeout time.Duration
	abandonGrace     time.Duration

	mu     sync.Mutex
	queues map[string][]job
}

func NewRouter() *Router {
	return &Router{
		handlers:         make(map[string]ActionHandler),
		actionTimeout:    defaultActionTimeout,
		inspectorTimeout: defaultInspectorTimeout,
		abandonGrace:     defaultAbandonGrace,
		queues:           make(map[string][]job),
	}
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
		r.enqueue(job{ev: ev, run: func(ctx context.Context, j job) error { return h.OnKeyUp(ctx, j.ev) }})
	case "dialRotate":
		if d, ok := h.(DialHandler); ok {
			r.enqueue(job{ev: ev, rotate: true, ticks: ev.Ticks(), run: func(ctx context.Context, j job) error {
				return d.OnDialRotate(ctx, j.ev, j.ticks)
			}})
		}
	case "dialDown", "touchTap":
		if d, ok := h.(DialHandler); ok {
			r.enqueue(job{ev: ev, run: func(ctx context.Context, j job) error { return d.OnDialPress(ctx, j.ev) }})
		}
	case "willAppear":
		if w, ok := h.(WillAppearHandler); ok {
			go r.run(ev, r.actionTimeout, func(ctx context.Context) error { return w.OnWillAppear(ctx, ev) })
		}
	case "sendToPlugin":
		if p, ok := h.(PluginMessageHandler); ok {
			go r.run(ev, r.inspectorTimeout, func(ctx context.Context) error { return p.OnSendToPlugin(ctx, ev) })
		}
	}
}

func (r *Router) enqueue(j job) {
	r.mu.Lock()
	defer r.mu.Unlock()
	queue, running := r.queues[j.ev.Context]
	// A dial sends ticks faster than DDC/CI can apply them, so pending rotations merge into one step.
	if last := len(queue) - 1; j.rotate && last >= 0 && queue[last].rotate {
		queue[last].ticks += j.ticks
		queue[last].ev = j.ev
		return
	}
	r.queues[j.ev.Context] = append(queue, j)
	if !running {
		go r.drain(j.ev.Context)
	}
	// The entry stays in the map, even when empty, until drain finds nothing left to run.
}

func (r *Router) drain(key string) {
	for {
		r.mu.Lock()
		queue := r.queues[key]
		if len(queue) == 0 {
			delete(r.queues, key)
			r.mu.Unlock()
			return
		}
		j := queue[0]
		r.queues[key] = queue[1:]
		r.mu.Unlock()

		r.run(j.ev, r.actionTimeout, func(ctx context.Context) error { return j.run(ctx, j) })
	}
}

// run gives up on a handler that ignores its deadline, because a DDC/CI call into a hung monitor cannot be cancelled.
func (r *Router) run(ev Event, timeout time.Duration, fn func(ctx context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- fn(ctx) }()

	abandon := time.NewTimer(timeout + r.abandonGrace)
	defer abandon.Stop()
	select {
	case err := <-done:
		if err != nil {
			log.Printf("%s %s: %v", ev.Event, ev.Action, err)
		}
	case <-abandon.C:
		log.Printf("%s %s: handler did not return, abandoned", ev.Event, ev.Action)
	}
}
