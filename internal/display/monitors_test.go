package display

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
)

func TestSortMonitorsByDisplayNumberThenID(t *testing.T) {
	mons := []Monitor{
		{ID: "c", DisplayNum: 2},
		{ID: "b", DisplayNum: 1},
		{ID: "a", DisplayNum: 2},
	}
	sortMonitors(mons)
	var ids []string
	for _, m := range mons {
		ids = append(ids, m.ID)
	}
	if want := []string{"b", "a", "c"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("got %v want %v", ids, want)
	}
}

func TestBuildMonitorsFillsInputsAndSorts(t *testing.T) {
	inputs := func(_ context.Context, id string) ([]InputPort, error) {
		return []InputPort{{Code: 0x11, Name: "HDMI 1 for " + id}}, nil
	}
	got, err := buildMonitors(context.Background(), []Monitor{
		{ID: "b", DisplayNum: 2},
		{ID: "a", DisplayNum: 1},
	}, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].ID != "a" || got[0].Inputs[0].Name != "HDMI 1 for a" || got[1].Inputs[0].Name != "HDMI 1 for b" {
		t.Fatalf("got %+v", got)
	}
}

func TestBuildMonitorsReturnsContextErrorBeforeReading(t *testing.T) {
	var calls atomic.Int32
	inputs := func(context.Context, string) ([]InputPort, error) {
		calls.Add(1)
		return nil, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := buildMonitors(ctx, []Monitor{{ID: "a"}}, inputs); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("inputs called %d times, want 0", calls.Load())
	}
}

func TestBuildMonitorsReturnsContextErrorAfterReading(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	inputs := func(context.Context, string) ([]InputPort, error) {
		cancel()
		return KnownInputPorts(), nil
	}
	if _, err := buildMonitors(ctx, []Monitor{{ID: "a"}}, inputs); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}
