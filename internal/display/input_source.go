package display

func readAndCacheInputSource(cache *Cache, monitorID string, read func() (uint32, error)) (uint32, error) {
	value, err := read()
	if err == nil && value > 0 {
		if cached, ok := cache.Get(monitorID); ok {
			cached.CurrentPort = value
			cache.Set(cached)
		}
	}
	return value, err
}
