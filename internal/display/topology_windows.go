//go:build windows

package display

import (
	"fmt"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func (m *windowsManager) resolve(id string) (monitorRef, error) {
	m.mu.Lock()
	ref, ok := m.byID[id]
	m.mu.Unlock()
	if ok {
		return ref, nil
	}
	if err := m.refreshTopology(); err != nil {
		return monitorRef{}, err
	}
	m.mu.Lock()
	ref, ok = m.byID[id]
	m.mu.Unlock()
	if !ok {
		return monitorRef{}, ErrMonitorNotFound
	}
	return ref, nil
}

// withRef retries once on a fresh topology, because handles and device names go stale
// when a display reconnects or wakes up.
func (m *windowsManager) withRef(id string, fn func(monitorRef) error) error {
	ref, err := m.resolve(id)
	if err != nil {
		return err
	}
	err = fn(ref)
	if err == nil {
		return nil
	}
	if m.refreshTopology() != nil {
		return err
	}
	fresh, resolveErr := m.resolve(id)
	if resolveErr != nil {
		return resolveErr
	}
	if fresh == ref {
		return err
	}
	return fn(fresh)
}

type enumeratedMonitor struct {
	hmon      windows.Handle
	dev       string
	isPrimary bool
	rect      windows.Rect
}

const monitorInfoFPrimary = 0x00000001

var (
	enumMu    sync.Mutex
	enumFound []enumeratedMonitor
	// Go never frees a callback slot, so the callback is created once and reused.
	enumCallback = sync.OnceValue(func() uintptr {
		return syscall.NewCallback(func(hmon windows.Handle, hdc windows.Handle, rect *windows.Rect, lparam uintptr) uintptr {
			var info MONITORINFOEX
			info.CbSize = uint32(unsafe.Sizeof(info))
			r, _, _ := procGetMonitorInfoW.Call(uintptr(hmon), uintptr(unsafe.Pointer(&info)))
			if r == 0 {
				return 1
			}
			enumFound = append(enumFound, enumeratedMonitor{
				hmon:      hmon,
				dev:       windows.UTF16ToString(info.SzDevice[:]),
				isPrimary: (info.DwFlags & monitorInfoFPrimary) != 0,
				rect:      info.RcMonitor,
			})
			return 1
		})
	})
)

func enumerateMonitors() []enumeratedMonitor {
	enumMu.Lock()
	defer enumMu.Unlock()
	enumFound = nil
	procEnumDisplayMonitors.Call(0, 0, enumCallback(), 0)
	found := enumFound
	enumFound = nil
	return found
}

func (m *windowsManager) refreshTopology() error {
	type dcTarget struct {
		adapterId windows.LUID
		targetId  uint32
	}
	gdiToDc := make(map[string]dcTarget)

	var numPath, numMode uint32
	r, _, _ := procGetDisplayConfigBufferSizes.Call(
		uintptr(qdcOnlyActivePaths),
		uintptr(unsafe.Pointer(&numPath)),
		uintptr(unsafe.Pointer(&numMode)),
	)
	if r == 0 && numPath > 0 {
		paths := make([]DISPLAYCONFIG_PATH_INFO, numPath)
		modes := make([]DISPLAYCONFIG_MODE_INFO, numMode)
		r2, _, _ := procQueryDisplayConfig.Call(
			uintptr(qdcOnlyActivePaths),
			uintptr(unsafe.Pointer(&numPath)),
			uintptr(unsafe.Pointer(&paths[0])),
			uintptr(unsafe.Pointer(&numMode)),
			uintptr(unsafe.Pointer(&modes[0])),
			0,
		)
		if r2 == 0 {
			for i := uint32(0); i < numPath; i++ {
				var src DISPLAYCONFIG_SOURCE_DEVICE_NAME
				src.Header.Type = displayConfigDeviceInfoGetSourceName
				src.Header.Size = uint32(unsafe.Sizeof(src))
				src.Header.AdapterId = paths[i].SourceInfo.AdapterId
				src.Header.Id = paths[i].SourceInfo.Id

				res, _, _ := procDisplayConfigGetDeviceInfo.Call(uintptr(unsafe.Pointer(&src)))
				if res == 0 {
					gdiName := windows.UTF16ToString(src.ViewGdiDeviceName[:])
					gdiToDc[gdiName] = dcTarget{
						adapterId: paths[i].TargetInfo.AdapterId,
						targetId:  paths[i].TargetInfo.Id,
					}
				}
			}
		}
	}

	next := make(map[string]monitorRef)
	for idx, p := range enumerateMonitors() {
		deviceName := windows.StringToUTF16Ptr(p.dev)
		var ddMon DISPLAY_DEVICE
		ddMon.Cb = uint32(unsafe.Sizeof(ddMon))

		procEnumDisplayDevicesW.Call(
			uintptr(unsafe.Pointer(deviceName)),
			0,
			uintptr(unsafe.Pointer(&ddMon)),
			0,
		)

		deviceID := windows.UTF16ToString(ddMon.DeviceID[:])
		rawName := windows.UTF16ToString(ddMon.DeviceString[:])

		name := queryEDIDModel(deviceID)

		if name == "" {
			_ = m.withPhysicalMonitors(p.hmon, func(mons []PHYSICAL_MONITOR) error {
				for _, pm := range mons {
					pmDesc := strings.TrimSpace(windows.UTF16ToString(pm.PhysicalMonitorDescription[:]))
					if pmDesc != "" && !strings.HasPrefix(strings.ToLower(pmDesc), "generic") {
						name = pmDesc
						break
					}
				}
				return nil
			})
		}

		if name == "" {
			if rawName != "" && !strings.HasPrefix(strings.ToLower(rawName), "generic") {
				name = rawName
			} else {
				name = fmt.Sprintf("Display %d", idx+1)
			}
		}

		dispNum := 0
		fmt.Sscanf(strings.TrimPrefix(p.dev, `\\.\DISPLAY`), "%d", &dispNum)
		if dispNum == 0 {
			dispNum = idx + 1
		}

		id := StableMonitorID(deviceID, "")
		if id == "" {
			id = fmt.Sprintf("disp_%s", strings.Trim(p.dev, `\.`))
		}

		ref := monitorRef{
			hmon:       p.hmon,
			device:     p.dev,
			name:       name,
			displayNum: dispNum,
			isPrimary:  p.isPrimary,
			rect:       p.rect,
		}

		if dc, ok := gdiToDc[p.dev]; ok {
			ref.adapterId = dc.adapterId
			ref.targetId = dc.targetId
			ref.hasHDR = true
		}

		next[id] = ref
	}

	m.mu.Lock()
	m.byID = next
	m.mu.Unlock()
	return nil
}
