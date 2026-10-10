//go:build windows

package display

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sys/windows"
)

type windowsManager struct {
	inputs *InputCache
	locks  *IDLocks
	mu     sync.Mutex
	hdrMu  sync.Mutex
	byID   map[string]monitorRef
}

type monitorRef struct {
	hmon       windows.Handle
	device     string
	name       string
	adapterId  windows.LUID
	targetId   uint32
	hasHDR     bool
	displayNum int
	isPrimary  bool
	rect       windows.Rect
}

func NewManager() Manager {
	m := &windowsManager{
		locks: NewIDLocks(),
		byID:  make(map[string]monitorRef),
	}
	m.inputs = NewInputCache(m.readCapabilitiesLocked, m.locks)
	return m
}

func (m *windowsManager) GetMonitors(ctx context.Context) ([]Monitor, error) {
	if err := m.refreshTopology(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	mons := make([]Monitor, 0, len(m.byID))
	for id, ref := range m.byID {
		mons = append(mons, Monitor{
			ID:         id,
			Name:       ref.name,
			DisplayNum: ref.displayNum,
			IsPrimary:  ref.isPrimary,
		})
	}
	m.mu.Unlock()

	return buildMonitors(ctx, mons, m.inputs.Inputs)
}

func (m *windowsManager) Identify(ctx context.Context) error {
	if err := m.refreshTopology(); err != nil {
		return err
	}
	m.mu.Lock()
	targets := make([]monitorOverlayTarget, 0, len(m.byID))
	for _, ref := range m.byID {
		targets = append(targets, monitorOverlayTarget{
			rect:       ref.rect,
			displayNum: ref.displayNum,
		})
	}
	m.mu.Unlock()

	go showIdentifyOverlays(targets)
	return nil
}

func (m *windowsManager) Sleep(ctx context.Context, monitorIDs []string) error {
	if len(monitorIDs) == 0 {
		return ErrNoMonitorsSelected
	}

	// The count decides between the desktop-wide power-off and per-monitor DDC/CI, so it must be current.
	_ = m.refreshTopology()
	m.mu.Lock()
	total := len(m.byID)
	m.mu.Unlock()

	if total > 0 && len(monitorIDs) >= total {
		procSendMessageW.Call(
			^uintptr(0),
			uintptr(wmSysCommand),
			uintptr(scMonitorPower),
			uintptr(monitorPowerOff),
		)
		return nil
	}

	var errs []error
	for _, id := range monitorIDs {
		if err := m.SetVCP(ctx, id, VCPPowerMode, powerModeOff); err != nil {
			errs = append(errs, fmt.Errorf("display: sleep %s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

func (m *windowsManager) Wake(ctx context.Context, monitorIDs []string) error {
	// A sleeping monitor often does not answer DDC/CI. The input event below wakes it anyway.
	for _, id := range monitorIDs {
		_ = m.SetVCP(ctx, id, VCPPowerMode, powerModeOn)
	}
	return sendWakeInput()
}

func sendWakeInput() error {
	procMouseEvent.Call(1, 0, 0, 0, 0)
	return nil
}
