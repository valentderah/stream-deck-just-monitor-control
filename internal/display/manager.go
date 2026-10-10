package display

import "context"

type Manager interface {
	GetMonitors(ctx context.Context) ([]Monitor, error)
	Identify(ctx context.Context) error
	SetVCP(ctx context.Context, monitorID string, code byte, value uint32) error
	GetVCP(ctx context.Context, monitorID string, code byte) (current, max uint32, err error)
	SetRefreshRate(ctx context.Context, monitorID string, rate RefreshRate) error
	ListRefreshRates(ctx context.Context, monitorID string) ([]RefreshRate, error)
	SetHDR(ctx context.Context, monitorID string, enabled bool) error
	GetHDR(ctx context.Context, monitorID string) (bool, error)
	Sleep(ctx context.Context, monitorIDs []string) error
	Wake(ctx context.Context, monitorIDs []string) error
}
