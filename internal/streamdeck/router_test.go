package streamdeck

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

const testAction = "com.valentderah.just-monitor-control.brightness"

type stubHandler struct {
	onKeyUp      func(ctx context.Context, ev Event) error
	onDialRotate func(ctx context.Context, ev Event, ticks int) error
	onDialPress  func(ctx context.Context, ev Event) error
}

func (s *stubHandler) OnKeyUp(ctx context.Context, ev Event) error {
	if s.onKeyUp != nil {
		return s.onKeyUp(ctx, ev)
	}
	return nil
}

func (s *stubHandler) OnDialRotate(ctx context.Context, ev Event, ticks int) error {
	if s.onDialRotate != nil {
		return s.onDialRotate(ctx, ev, ticks)
	}
	return nil
}

func (s *stubHandler) OnDialPress(ctx context.Context, ev Event) error {
	if s.onDialPress != nil {
		return s.onDialPress(ctx, ev)
	}
	return nil
}

type keyOnlyHandler struct{ keys chan struct{} }

func (k keyOnlyHandler) OnKeyUp(context.Context, Event) error {
	k.keys <- struct{}{}
	return nil
}

func keyUp(context string) Event {
	return Event{Event: "keyUp", Action: testAction, Context: context}
}

func dialRotate(context string, ticks int) Event {
	payload, _ := json.Marshal(map[string]any{"ticks": ticks, "controller": ControllerEncoder})
	return Event{Event: "dialRotate", Action: testAction, Context: context, Payload: payload}
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func TestRouterHandleDoesNotBlockOnHandler(t *testing.T) {
	r := NewRouter()
	started := make(chan struct{})
	release := make(chan struct{})
	r.Register(testAction, &stubHandler{
		onKeyUp: func(ctx context.Context, ev Event) error {
			close(started)
			<-release
			return nil
		},
	})
	done := make(chan struct{})
	go func() {
		r.Handle(keyUp("c"))
		close(done)
	}()
	waitFor(t, done, "Handle to return")
	waitFor(t, started, "handler to start")
	close(release)
}

func TestRouterRunsOneContextInOrder(t *testing.T) {
	r := NewRouter()
	var mu sync.Mutex
	running, maxRunning := 0, 0
	finished := make(chan struct{}, 3)
	r.Register(testAction, &stubHandler{
		onKeyUp: func(ctx context.Context, ev Event) error {
			mu.Lock()
			running++
			maxRunning = max(maxRunning, running)
			mu.Unlock()
			time.Sleep(20 * time.Millisecond)
			mu.Lock()
			running--
			mu.Unlock()
			finished <- struct{}{}
			return nil
		},
	})
	for range 3 {
		r.Handle(keyUp("c"))
	}
	for range 3 {
		waitFor(t, finished, "queued key presses")
	}
	if maxRunning != 1 {
		t.Fatalf("%d handlers ran at once on one context, want 1", maxRunning)
	}
}

func TestRouterRunsContextsInParallel(t *testing.T) {
	r := NewRouter()
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	r.Register(testAction, &stubHandler{
		onKeyUp: func(ctx context.Context, ev Event) error {
			started <- struct{}{}
			<-release
			return nil
		},
	})
	r.Handle(keyUp("a"))
	r.Handle(keyUp("b"))
	waitFor(t, started, "first context")
	waitFor(t, started, "second context")
	close(release)
}

func TestRouterMergesPendingRotations(t *testing.T) {
	r := NewRouter()
	entered := make(chan struct{})
	release := make(chan struct{})
	got := make(chan int, 4)
	first := true
	r.Register(testAction, &stubHandler{
		onDialRotate: func(ctx context.Context, ev Event, ticks int) error {
			if first {
				first = false
				close(entered)
				<-release
			}
			got <- ticks
			return nil
		},
	})
	r.Handle(dialRotate("c", 1))
	waitFor(t, entered, "first rotation")
	r.Handle(dialRotate("c", 2))
	r.Handle(dialRotate("c", 3))
	r.Handle(dialRotate("c", -1))
	close(release)

	for _, want := range []int{1, 4} {
		select {
		case ticks := <-got:
			if ticks != want {
				t.Fatalf("handler got %d ticks, want %d", ticks, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %d ticks", want)
		}
	}
	select {
	case ticks := <-got:
		t.Fatalf("unexpected extra rotation of %d ticks", ticks)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRouterDoesNotMergeRotationAcrossPress(t *testing.T) {
	r := NewRouter()
	entered := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	var calls []string
	done := make(chan struct{}, 4)
	record := func(name string) {
		mu.Lock()
		calls = append(calls, name)
		mu.Unlock()
		done <- struct{}{}
	}
	r.Register(testAction, &stubHandler{
		onKeyUp: func(ctx context.Context, ev Event) error {
			close(entered)
			<-release
			return nil
		},
		onDialRotate: func(ctx context.Context, ev Event, ticks int) error {
			record("rotate")
			return nil
		},
		onDialPress: func(ctx context.Context, ev Event) error {
			record("press")
			return nil
		},
	})
	r.Handle(keyUp("c"))
	waitFor(t, entered, "blocking handler")
	r.Handle(dialRotate("c", 1))
	r.Handle(Event{Event: "dialDown", Action: testAction, Context: "c"})
	r.Handle(dialRotate("c", 1))
	close(release)
	for range 3 {
		waitFor(t, done, "queued dial events")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 3 || calls[0] != "rotate" || calls[1] != "press" || calls[2] != "rotate" {
		t.Fatalf("calls = %v, want [rotate press rotate]", calls)
	}
}

func TestRouterIgnoresDialEventsForKeyOnlyAction(t *testing.T) {
	r := NewRouter()
	h := keyOnlyHandler{keys: make(chan struct{}, 1)}
	r.Register(testAction, h)
	r.Handle(dialRotate("c", 1))
	r.Handle(Event{Event: "dialDown", Action: testAction, Context: "c"})
	r.Handle(Event{Event: "willAppear", Action: testAction, Context: "c"})
	r.Handle(Event{Event: "sendToPlugin", Action: testAction, Context: "c"})
	r.Handle(keyUp("c"))
	waitFor(t, h.keys, "key press")
}

func TestRouterAbandonsHandlerThatIgnoresDeadline(t *testing.T) {
	r := NewRouter()
	r.actionTimeout = 10 * time.Millisecond
	r.abandonGrace = 10 * time.Millisecond
	release := make(chan struct{})
	defer close(release)
	second := make(chan struct{})
	calls := 0
	r.Register(testAction, &stubHandler{
		onKeyUp: func(ctx context.Context, ev Event) error {
			calls++
			if calls == 1 {
				<-release
				return nil
			}
			close(second)
			return nil
		},
	})
	r.Handle(keyUp("c"))
	r.Handle(keyUp("c"))
	waitFor(t, second, "the press queued behind a hung handler")
}

func TestEventControllerAndTicks(t *testing.T) {
	cases := []struct {
		payload    string
		controller string
		ticks      int
	}{
		{``, ControllerKeypad, 0},
		{`{"settings":{}}`, ControllerKeypad, 0},
		{`{"controller":"Keypad"}`, ControllerKeypad, 0},
		{`{"controller":"Encoder","ticks":-3}`, ControllerEncoder, -3},
	}
	for _, c := range cases {
		ev := Event{Payload: json.RawMessage(c.payload)}
		if ev.Controller() != c.controller || ev.Ticks() != c.ticks {
			t.Errorf("%q: got %s and %d, want %s and %d", c.payload, ev.Controller(), ev.Ticks(), c.controller, c.ticks)
		}
	}
}
