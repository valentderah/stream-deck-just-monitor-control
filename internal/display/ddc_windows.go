//go:build windows

package display

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func (m *windowsManager) SetVCP(ctx context.Context, monitorID string, code byte, value uint32) error {
	return m.withDDC(ctx, monitorID, func(hmon windows.Handle) error {
		return m.setVCPFeatureLocked(hmon, code, value)
	})
}

func (m *windowsManager) GetVCP(ctx context.Context, monitorID string, code byte) (current, max uint32, err error) {
	err = m.withDDC(ctx, monitorID, func(hmon windows.Handle) error {
		var err error
		current, max, err = m.getVCPFeatureLocked(hmon, code)
		return err
	})
	return current, max, err
}

// withDDC holds the monitor's lock for the whole exchange, because DDC/CI cannot interleave commands.
func (m *windowsManager) withDDC(ctx context.Context, monitorID string, fn func(hmon windows.Handle) error) error {
	unlock, err := m.locks.Lock(ctx, monitorID)
	if err != nil {
		return err
	}
	defer unlock()
	return m.withRef(monitorID, func(ref monitorRef) error { return fn(ref.hmon) })
}

func (m *windowsManager) withPhysicalMonitors(hmon windows.Handle, fn func([]PHYSICAL_MONITOR) error) error {
	var count uint32
	r, _, err := procGetNumberOfPhysicalMonitorsFromHMONITOR.Call(uintptr(hmon), uintptr(unsafe.Pointer(&count)))
	if r == 0 || count == 0 {
		return fmt.Errorf("display: no physical monitors: %w", err)
	}
	mons := make([]PHYSICAL_MONITOR, count)
	r, _, err = procGetPhysicalMonitorsFromHMONITOR.Call(
		uintptr(hmon),
		uintptr(count),
		uintptr(unsafe.Pointer(&mons[0])),
	)
	if r == 0 {
		return fmt.Errorf("display: GetPhysicalMonitors failed: %w", err)
	}
	defer procDestroyPhysicalMonitors.Call(uintptr(count), uintptr(unsafe.Pointer(&mons[0])))
	return fn(mons)
}

func (m *windowsManager) readCapabilitiesLocked(monitorID string) (string, error) {
	var caps string
	err := m.withRef(monitorID, func(ref monitorRef) error {
		return m.withPhysicalMonitors(ref.hmon, func(mons []PHYSICAL_MONITOR) error {
			var last error
			for _, pm := range mons {
				var length uint32
				r1, _, callErr := procGetCapabilitiesStringLength.Call(
					uintptr(pm.HPhysicalMonitor),
					uintptr(unsafe.Pointer(&length)),
				)
				time.Sleep(ddcInterCommandDelay)
				if r1 == 0 || length == 0 {
					last = callErr
					continue
				}
				buf := make([]byte, length)
				r1, _, callErr = procCapabilitiesRequestAndCapabilitiesReply.Call(
					uintptr(pm.HPhysicalMonitor),
					uintptr(unsafe.Pointer(&buf[0])),
					uintptr(length),
				)
				time.Sleep(ddcInterCommandDelay)
				if r1 == 0 {
					last = callErr
					continue
				}
				caps = windows.ByteSliceToString(buf)
				return nil
			}
			return last
		})
	})
	return caps, err
}

func (m *windowsManager) getVCPFeatureLocked(hmon windows.Handle, code byte) (current, max uint32, err error) {
	err = m.withPhysicalMonitors(hmon, func(mons []PHYSICAL_MONITOR) error {
		var last error
		for _, pm := range mons {
			var curVal, maxVal uint32
			var vcpType uint32
			r1, _, callErr := procGetVCPFeatureAndVCPFeatureReply.Call(
				uintptr(pm.HPhysicalMonitor),
				uintptr(code),
				uintptr(unsafe.Pointer(&vcpType)),
				uintptr(unsafe.Pointer(&curVal)),
				uintptr(unsafe.Pointer(&maxVal)),
			)
			time.Sleep(ddcInterCommandDelay)
			if r1 == 0 {
				last = callErr
				continue
			}
			current, max = curVal, maxVal
			return nil
		}
		return last
	})
	return current, max, err
}

func (m *windowsManager) setVCPFeatureLocked(hmon windows.Handle, code byte, value uint32) error {
	return m.withPhysicalMonitors(hmon, func(mons []PHYSICAL_MONITOR) error {
		var last error
		for _, pm := range mons {
			r1, _, callErr := procSetVCPFeature.Call(
				uintptr(pm.HPhysicalMonitor),
				uintptr(code),
				uintptr(value),
			)
			time.Sleep(ddcInterCommandDelay)
			if r1 == 0 {
				last = callErr
				continue
			}
			return nil
		}
		return last
	})
}
