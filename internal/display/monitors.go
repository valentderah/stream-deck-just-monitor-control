package display

import (
	"cmp"
	"context"
	"slices"
	"sync"
)

func sortMonitors(mons []Monitor) {
	slices.SortFunc(mons, func(a, b Monitor) int {
		if c := cmp.Compare(a.DisplayNum, b.DisplayNum); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
}

func buildMonitors(ctx context.Context, mons []Monitor, inputs func(context.Context, string) ([]InputPort, error)) ([]Monitor, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	for i := range mons {
		wg.Go(func() {
			ports, err := inputs(ctx, mons[i].ID)
			if err != nil {
				ports = KnownInputPorts()
			}
			mons[i].Inputs = ports
		})
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sortMonitors(mons)
	return mons, nil
}
