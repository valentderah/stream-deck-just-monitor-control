package display

import (
	"context"
	"slices"
	"sync"
)

// CapabilitiesReader is called with the monitor's IDLocks lock held and must not take it.
type CapabilitiesReader func(monitorID string) (string, error)

type InputCache struct {
	read  CapabilitiesReader
	locks *IDLocks
	mu    sync.RWMutex
	ports map[string][]InputPort
}

func NewInputCache(read CapabilitiesReader, locks *IDLocks) *InputCache {
	return &InputCache{read: read, locks: locks, ports: make(map[string][]InputPort)}
}

func (c *InputCache) Inputs(ctx context.Context, monitorID string) ([]InputPort, error) {
	if ports, ok := c.cached(monitorID); ok {
		return ports, nil
	}
	unlock, err := c.locks.Lock(ctx, monitorID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if ports, ok := c.cached(monitorID); ok {
		return ports, nil
	}
	caps, err := c.read(monitorID)
	if err != nil {
		return KnownInputPorts(), nil
	}
	ports, ok := InputPortsFromCapabilities(caps)
	if ok {
		c.mu.Lock()
		c.ports[monitorID] = ports
		c.mu.Unlock()
	}
	return slices.Clone(ports), nil
}

func (c *InputCache) cached(monitorID string) ([]InputPort, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ports, ok := c.ports[monitorID]
	return slices.Clone(ports), ok
}
