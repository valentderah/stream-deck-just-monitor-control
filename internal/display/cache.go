package display

import "sync"

type Cache struct {
	mu sync.RWMutex
	m  map[string]Monitor
}

func NewCache() *Cache {
	return &Cache{m: make(map[string]Monitor)}
}

func (c *Cache) Get(id string) (Monitor, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m, ok := c.m[id]
	return m, ok
}

func (c *Cache) Set(mon Monitor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[mon.ID] = mon
}

func (c *Cache) Invalidate(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, id)
}

func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[string]Monitor)
}
