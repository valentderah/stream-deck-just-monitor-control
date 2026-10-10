package display

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIDLocksSequential(t *testing.T) {
	l := NewIDLocks()
	for range 2 {
		unlock, err := l.Lock(context.Background(), "a")
		if err != nil {
			t.Fatal(err)
		}
		unlock()
	}
}

func TestIDLocksKeepsIDsIndependent(t *testing.T) {
	l := NewIDLocks()
	unlockA, err := l.Lock(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	defer unlockA()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	unlockB, err := l.Lock(ctx, "b")
	if err != nil {
		t.Fatalf("lock on another id blocked: %v", err)
	}
	unlockB()
}

func TestIDLocksGivesUpWhenContextEnds(t *testing.T) {
	l := NewIDLocks()
	unlock, err := l.Lock(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := l.Lock(ctx, "a"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want context.DeadlineExceeded", err)
	}
	unlock()
	unlock, err = l.Lock(context.Background(), "a")
	if err != nil {
		t.Fatalf("lock stayed held after a waiter gave up: %v", err)
	}
	unlock()
}
