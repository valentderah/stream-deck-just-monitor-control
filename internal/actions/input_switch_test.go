package actions

import (
	"context"
	"errors"
	"testing"
)

func TestWaitForInputSourceStopsAfterMatch(t *testing.T) {
	reads := 0
	read := func(context.Context) (uint32, error) {
		reads++
		if reads == 1 {
			return 15, nil
		}
		return 17, nil
	}

	err := waitForInputSource(context.Background(), 17, 3, 0, read)
	if err != nil {
		t.Fatalf("waitForInputSource returned error: %v", err)
	}
	if reads != 2 {
		t.Fatalf("read called %d times, want 2", reads)
	}
}

func TestWaitForInputSourceReturnsNotConfirmedAfterAttempts(t *testing.T) {
	reads := 0
	readErr := errors.New("read failed")
	read := func(context.Context) (uint32, error) {
		reads++
		if reads == 2 {
			return 0, readErr
		}
		return 15, nil
	}

	err := waitForInputSource(context.Background(), 17, 3, 0, read)
	if !errors.Is(err, errInputSourceNotConfirmed) {
		t.Fatalf("waitForInputSource returned %v, want %v", err, errInputSourceNotConfirmed)
	}
	if reads != 3 {
		t.Fatalf("read called %d times, want 3", reads)
	}
}
