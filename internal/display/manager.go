package display

import "context"

type Manager interface {
	GetMonitors(ctx context.Context) ([]Monitor, error)
	Identify(ctx context.Context) error
	SetBrightness(ctx context.Context, monitorID string, value uint32) error
	GetBrightness(ctx context.Context, monitorID string) (uint32, error)
	SetInputSource(ctx context.Context, monitorID string, source uint32) error
	GetInputSource(ctx context.Context, monitorID string) (uint32, error)
	SetRefreshRate(ctx context.Context, monitorID string, rate RefreshRate) error
	ListRefreshRates(ctx context.Context, monitorID string) ([]RefreshRate, error)
	SetHDR(ctx context.Context, monitorID string, enabled bool) error
	GetHDR(ctx context.Context, monitorID string) (bool, error)
	Sleep(ctx context.Context, monitorIDs []string) error
	Wake(ctx context.Context, monitorIDs []string) error
	SetVCP(ctx context.Context, monitorID string, code byte, value uint32) error
	GetVCP(ctx context.Context, monitorID string, code byte) (current, max uint32, err error)
	SubscribeChanges(ctx context.Context) (<-chan struct{}, error)
}
