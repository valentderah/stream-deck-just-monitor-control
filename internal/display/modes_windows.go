//go:build windows

package display

import (
	"context"
	"fmt"
	"sort"
	"unsafe"

	"golang.org/x/sys/windows"
)

func (m *windowsManager) ListRefreshRates(ctx context.Context, monitorID string) ([]RefreshRate, error) {
	var rates []RefreshRate
	err := m.withRef(monitorID, func(ref monitorRef) error {
		var err error
		rates, err = listRefreshRates(ref.device)
		return err
	})
	return rates, err
}

func (m *windowsManager) SetRefreshRate(ctx context.Context, monitorID string, rate RefreshRate) error {
	return m.withRef(monitorID, func(ref monitorRef) error {
		return setRefreshRate(ref.device, rate)
	})
}

func (m *windowsManager) GetHDR(ctx context.Context, monitorID string) (bool, error) {
	var enabled bool
	err := m.withRef(monitorID, func(ref monitorRef) error {
		var err error
		enabled, err = getHDR(ref)
		return err
	})
	return enabled, err
}

func (m *windowsManager) SetHDR(ctx context.Context, monitorID string, enabled bool) error {
	// DisplayConfig changes are applied one at a time.
	m.hdrMu.Lock()
	defer m.hdrMu.Unlock()
	return m.withRef(monitorID, func(ref monitorRef) error {
		return setHDR(ref, enabled)
	})
}

func listRefreshRates(deviceName string) ([]RefreshRate, error) {
	device := windows.StringToUTF16Ptr(deviceName)
	seen := map[RefreshRate]struct{}{}
	var out []RefreshRate
	var modeIdx uint32
	for {
		var dm DEVMODEW
		dm.DmSize = uint16(unsafe.Sizeof(dm))
		r, _, _ := procEnumDisplaySettingsW.Call(
			uintptr(unsafe.Pointer(device)),
			uintptr(modeIdx),
			uintptr(unsafe.Pointer(&dm)),
		)
		if r == 0 {
			break
		}
		if dm.DmDisplayFrequency > 1 {
			rr := RefreshRate{Numerator: dm.DmDisplayFrequency, Denominator: 1}
			if _, ok := seen[rr]; !ok {
				seen[rr] = struct{}{}
				out = append(out, rr)
			}
		}
		modeIdx++
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("display: no refresh rates found for %s", deviceName)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Hertz() < out[j].Hertz()
	})

	return out, nil
}

func setRefreshRate(deviceName string, rate RefreshRate) error {
	device := windows.StringToUTF16Ptr(deviceName)
	var dm DEVMODEW
	dm.DmSize = uint16(unsafe.Sizeof(dm))
	r, _, _ := procEnumDisplaySettingsW.Call(
		uintptr(unsafe.Pointer(device)),
		uintptr(enumCurrentSettings),
		uintptr(unsafe.Pointer(&dm)),
	)
	if r == 0 {
		return fmt.Errorf("display: EnumDisplaySettings current failed")
	}

	hz := rate.Numerator
	if rate.Denominator > 1 {
		hz = uint32(rate.Hertz() + 0.5)
	}
	dm.DmFields |= 0x00400000
	dm.DmDisplayFrequency = hz

	ret, _, callErr := procChangeDisplaySettingsExW.Call(
		uintptr(unsafe.Pointer(device)),
		uintptr(unsafe.Pointer(&dm)),
		0,
		uintptr(cdsUpdateregistry),
		0,
	)
	if int32(ret) != dispChangeSuccessful {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("display: ChangeDisplaySettingsEx: %w", callErr)
		}
		return fmt.Errorf("display: ChangeDisplaySettingsEx failed code %d", ret)
	}
	return nil
}

func getHDR(ref monitorRef) (bool, error) {
	if !ref.hasHDR {
		return false, fmt.Errorf("display: no DisplayConfig targetId found for %s", ref.device)
	}

	var info DISPLAYCONFIG_GET_ADVANCED_COLOR_INFO
	info.Header.Type = displayConfigDeviceInfoGetAdvancedColorInfo
	info.Header.Size = uint32(unsafe.Sizeof(info))
	info.Header.AdapterId = ref.adapterId
	info.Header.Id = ref.targetId

	r, _, errCall := procDisplayConfigGetDeviceInfo.Call(uintptr(unsafe.Pointer(&info)))
	if r != 0 {
		return false, fmt.Errorf("display: GetAdvancedColorInfo failed: %v", errCall)
	}
	enabled := (info.Value & 0x2) != 0
	return enabled, nil
}

func setHDR(ref monitorRef, enabled bool) error {
	if !ref.hasHDR {
		return fmt.Errorf("display: no DisplayConfig targetId found for %s", ref.device)
	}

	var state DISPLAYCONFIG_SET_ADVANCED_COLOR_STATE
	state.Header.Type = displayConfigDeviceInfoSetAdvancedColorState
	state.Header.Size = uint32(unsafe.Sizeof(state))
	state.Header.AdapterId = ref.adapterId
	state.Header.Id = ref.targetId
	if enabled {
		state.Value = 1
	} else {
		state.Value = 0
	}

	r, _, errCall := procDisplayConfigSetDeviceInfo.Call(uintptr(unsafe.Pointer(&state)))
	if r != 0 {
		return fmt.Errorf("display: SetAdvancedColorState failed code %d: %v", r, errCall)
	}
	return nil
}
