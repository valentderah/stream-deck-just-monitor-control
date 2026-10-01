package display

import "testing"

func TestReadAndCacheInputSourceOverridesStaleCachedPort(t *testing.T) {
	cache := NewCache()
	cache.Set(Monitor{ID: "monitor-1", CurrentPort: 0x0F})

	readCalled := false
	got, err := readAndCacheInputSource(cache, "monitor-1", func() (uint32, error) {
		readCalled = true
		return 0x11, nil
	})
	if err != nil {
		t.Fatalf("readAndCacheInputSource returned error: %v", err)
	}
	if !readCalled {
		t.Fatal("hardware read callback was not called")
	}
	if got != 0x11 {
		t.Fatalf("readAndCacheInputSource = %#x, want %#x", got, uint32(0x11))
	}

	cached, ok := cache.Get("monitor-1")
	if !ok {
		t.Fatal("monitor missing from cache")
	}
	if cached.CurrentPort != 0x11 {
		t.Fatalf("cached CurrentPort = %#x, want %#x", cached.CurrentPort, uint32(0x11))
	}
}
