//go:build !windows

package display

import "context"

type stubManager struct{}

func NewManager() Manager {
	return stubManager{}
}

func (stubManager) GetMonitors(context.Context) ([]Monitor, error) {
	return nil, ErrUnsupported
}
func (stubManager) Identify(context.Context) error { return ErrUnsupported }
func (stubManager) SetBrightness(context.Context, string, uint32) error {
	return ErrUnsupported
}
func (stubManager) GetBrightness(context.Context, string) (uint32, error) {
	return 0, ErrUnsupported
}
func (stubManager) SetInputSource(context.Context, string, uint32) error {
	return ErrUnsupported
}
func (stubManager) GetInputSource(context.Context, string) (uint32, error) {
	return 0, ErrUnsupported
}
func (stubManager) SetRefreshRate(context.Context, string, RefreshRate) error {
	return ErrUnsupported
}
func (stubManager) ListRefreshRates(context.Context, string) ([]RefreshRate, error) {
	return nil, ErrUnsupported
}
func (stubManager) SetHDR(context.Context, string, bool) error { return ErrUnsupported }
func (stubManager) GetHDR(context.Context, string) (bool, error) {
	return false, ErrUnsupported
}
func (stubManager) Sleep(context.Context, []string) error { return ErrUnsupported }
func (stubManager) Wake(context.Context, []string) error  { return ErrUnsupported }
func (stubManager) SetVCP(context.Context, string, byte, uint32) error {
	return ErrUnsupported
}
func (stubManager) GetVCP(context.Context, string, byte) (uint32, uint32, error) {
	return 0, 0, ErrUnsupported
}
func (stubManager) SubscribeChanges(context.Context) (<-chan struct{}, error) {
	return nil, ErrUnsupported
}
