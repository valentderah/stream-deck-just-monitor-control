package display

import "testing"

func TestCacheSetGetAndInvalidate(t *testing.T) {
	c := NewCache()
	c.Set(Monitor{ID: "a", Brightness: 40})
	m, ok := c.Get("a")
	if !ok || m.Brightness != 40 {
		t.Fatalf("got %+v ok=%v", m, ok)
	}
	c.Invalidate("a")
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected miss after invalidate")
	}
	c.Set(Monitor{ID: "b", Brightness: 1})
	c.InvalidateAll()
	if _, ok := c.Get("b"); ok {
		t.Fatal("expected miss after InvalidateAll")
	}
}
