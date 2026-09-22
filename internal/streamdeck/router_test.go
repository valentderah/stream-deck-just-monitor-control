package streamdeck

import (
	"context"
	"testing"
	"time"
)

type stubHandler struct {
	onKeyUp func(ctx context.Context, ev Event) error
}

func (s *stubHandler) OnKeyUp(ctx context.Context, ev Event) error {
	if s.onKeyUp != nil {
		return s.onKeyUp(ctx, ev)
	}
	return nil
}
func (s *stubHandler) OnWillAppear(context.Context, Event) error                { return nil }
func (s *stubHandler) OnPropertyInspectorDidAppear(context.Context, Event) error { return nil }
func (s *stubHandler) OnSendToPlugin(context.Context, Event) error               { return nil }

func TestRouterHandleDoesNotBlockOnHandler(t *testing.T) {
	r := NewRouter()
	started := make(chan struct{})
	release := make(chan struct{})
	r.Register("com.valentderah.just-monitor-control.brightness", &stubHandler{
		onKeyUp: func(ctx context.Context, ev Event) error {
			close(started)
			<-release
			return nil
		},
	})
	done := make(chan struct{})
	go func() {
		r.Handle(Event{Event: "keyUp", Action: "com.valentderah.just-monitor-control.brightness", Context: "c"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Handle blocked on handler")
	}
	select {
	case <-started:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("handler did not start")
	}
	close(release)
}
