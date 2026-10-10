package display

import (
	"context"
	"sync"
)

type IDLocks struct {
	mu    sync.Mutex
	locks map[string]chan struct{}
}

func NewIDLocks() *IDLocks {
	return &IDLocks{locks: make(map[string]chan struct{})}
}

// Lock gives up when ctx ends, so a holder stuck in a monitor call does not block later callers forever.
func (l *IDLocks) Lock(ctx context.Context, id string) (unlock func(), err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	l.mu.Lock()
	ch, ok := l.locks[id]
	if !ok {
		ch = make(chan struct{}, 1)
		l.locks[id] = ch
	}
	l.mu.Unlock()

	select {
	case ch <- struct{}{}:
		return func() { <-ch }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
