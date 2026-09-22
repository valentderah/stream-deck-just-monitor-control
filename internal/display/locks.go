package display

import "sync"

type IDLocks struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewIDLocks() *IDLocks {
	return &IDLocks{locks: make(map[string]*sync.Mutex)}
}

func (l *IDLocks) Lock(id string) func() {
	l.mu.Lock()
	m, ok := l.locks[id]
	if !ok {
		m = &sync.Mutex{}
		l.locks[id] = m
	}
	l.mu.Unlock()
	m.Lock()
	return m.Unlock
}
