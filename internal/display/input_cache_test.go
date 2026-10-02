package display

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const capsWithInputs = `vcp(10 60(11 0F))`

func countingReader(caps map[string]string, err error) (CapabilitiesReader, *atomic.Int32) {
	var calls atomic.Int32
	return func(id string) (string, error) {
		calls.Add(1)
		return caps[id], err
	}, &calls
}

func TestInputCacheCachesSuccessfulRead(t *testing.T) {
	read, calls := countingReader(map[string]string{"a": capsWithInputs}, nil)
	c := NewInputCache(read, NewIDLocks())
	first, err := c.Inputs(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := c.Inputs(context.Background(), "a")
	if calls.Load() != 1 {
		t.Fatalf("reader called %d times, want 1", calls.Load())
	}
	want := []InputPort{{Code: 0x11, Name: "HDMI 1"}, {Code: 0x0F, Name: "DisplayPort 1"}}
	if !reflect.DeepEqual(first, want) || !reflect.DeepEqual(second, want) {
		t.Fatalf("got %v and %v, want %v", first, second, want)
	}
}

func TestInputCacheDoesNotCacheFailures(t *testing.T) {
	cases := map[string]struct {
		caps map[string]string
		err  error
	}{
		"read error":   {err: errors.New("ddc failed")},
		"empty string": {caps: map[string]string{"a": ""}},
		"no 60":        {caps: map[string]string{"a": "vcp(10 12)"}},
		"malformed 60": {caps: map[string]string{"a": "vcp(60(0F zz))"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			read, calls := countingReader(tc.caps, tc.err)
			c := NewInputCache(read, NewIDLocks())
			for range 2 {
				got, err := c.Inputs(context.Background(), "a")
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, KnownInputPorts()) {
					t.Fatalf("got %v, want fallback", got)
				}
			}
			if calls.Load() != 2 {
				t.Fatalf("reader called %d times, want 2", calls.Load())
			}
		})
	}
}

func TestInputCacheKeepsMonitorsIndependent(t *testing.T) {
	read, calls := countingReader(map[string]string{
		"a": `vcp(60(11))`,
		"b": `vcp(60(0F))`,
	}, nil)
	c := NewInputCache(read, NewIDLocks())
	a, _ := c.Inputs(context.Background(), "a")
	b, _ := c.Inputs(context.Background(), "b")
	_, _ = c.Inputs(context.Background(), "a")
	if calls.Load() != 2 {
		t.Fatalf("reader called %d times, want 2", calls.Load())
	}
	if a[0].Code != 0x11 || b[0].Code != 0x0F {
		t.Fatalf("got a=%v b=%v", a, b)
	}
}

func TestInputCacheConcurrentCallsReadOnce(t *testing.T) {
	var calls atomic.Int32
	entered := make(chan struct{})
	release := make(chan struct{})
	read := func(string) (string, error) {
		if calls.Add(1) == 1 {
			close(entered)
		}
		<-release
		return capsWithInputs, nil
	}
	c := NewInputCache(read, NewIDLocks())

	var wg sync.WaitGroup
	wg.Go(func() { _, _ = c.Inputs(context.Background(), "a") })
	<-entered
	wg.Go(func() { _, _ = c.Inputs(context.Background(), "a") })
	// Gives the second call time to miss the cache and block on the monitor lock.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("reader called %d times, want 1", calls.Load())
	}
}

func TestInputCacheReturnsContextErrorBeforeRead(t *testing.T) {
	read, calls := countingReader(map[string]string{"a": capsWithInputs}, nil)
	c := NewInputCache(read, NewIDLocks())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Inputs(ctx, "a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("reader called %d times, want 0", calls.Load())
	}
}

func TestInputCacheKeepsReadThatFinishesAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	read := func(string) (string, error) {
		calls.Add(1)
		cancel()
		return capsWithInputs, nil
	}
	c := NewInputCache(read, NewIDLocks())
	if _, err := c.Inputs(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Inputs(context.Background(), "a"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("reader called %d times, want 1", calls.Load())
	}
}
