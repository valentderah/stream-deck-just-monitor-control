//go:build windows

package display

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	VCPBrightness        = 0x10
	VCPInput             = 0x60
	VCPPowerMode         = 0xD6
	ddcInterCommandDelay = 40 * time.Millisecond
	wmSysCommand         = 0x0112
	scMonitorPower       = 0xF170
	monitorPowerOff      = 2
	enumCurrentSettings  = uint32(0xFFFFFFFF)
	cdsUpdateregistry    = 0x00000001
	dispChangeSuccessful = 0

	qdcOnlyActivePaths                           = 0x00000002
	displayConfigDeviceInfoGetSourceName         = 1
	displayConfigDeviceInfoGetAdvancedColorInfo  = 9
	displayConfigDeviceInfoSetAdvancedColorState = 10
)

var (
	user32                          = windows.NewLazySystemDLL("user32.dll")
	dxva2                           = windows.NewLazySystemDLL("dxva2.dll")
	procEnumDisplayMonitors         = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW             = user32.NewProc("GetMonitorInfoW")
	procEnumDisplayDevicesW         = user32.NewProc("EnumDisplayDevicesW")
	procEnumDisplaySettingsW        = user32.NewProc("EnumDisplaySettingsW")
	procChangeDisplaySettingsExW    = user32.NewProc("ChangeDisplaySettingsExW")
	procSendMessageW                = user32.NewProc("SendMessageW")
	procMouseEvent                  = user32.NewProc("mouse_event")
	procGetDisplayConfigBufferSizes = user32.NewProc("GetDisplayConfigBufferSizes")
	procQueryDisplayConfig          = user32.NewProc("QueryDisplayConfig")
	procDisplayConfigGetDeviceInfo  = user32.NewProc("DisplayConfigGetDeviceInfo")
	procDisplayConfigSetDeviceInfo  = user32.NewProc("DisplayConfigSetDeviceInfo")

	procGetNumberOfPhysicalMonitorsFromHMONITOR = dxva2.NewProc("GetNumberOfPhysicalMonitorsFromHMONITOR")
	procGetPhysicalMonitorsFromHMONITOR         = dxva2.NewProc("GetPhysicalMonitorsFromHMONITOR")
	procDestroyPhysicalMonitors                 = dxva2.NewProc("DestroyPhysicalMonitors")
	procGetVCPFeatureAndVCPFeatureReply         = dxva2.NewProc("GetVCPFeatureAndVCPFeatureReply")
	procSetVCPFeature                           = dxva2.NewProc("SetVCPFeature")
)

type PHYSICAL_MONITOR struct {
	HPhysicalMonitor           windows.Handle
	PhysicalMonitorDescription [128]uint16
}

type MONITORINFOEX struct {
	CbSize    uint32
	RcMonitor windows.Rect
	RcWork    windows.Rect
	DwFlags   uint32
	SzDevice  [32]uint16
}

type DISPLAY_DEVICE struct {
	Cb           uint32
	DeviceName   [32]uint16
	DeviceString [128]uint16
	StateFlags   uint32
	DeviceID     [128]uint16
	DeviceKey    [128]uint16
}

type DEVMODEW struct {
	DmDeviceName         [32]uint16
	DmSpecVersion        uint16
	DmDriverVersion      uint16
	DmSize               uint16
	DmDriverExtra        uint16
	DmFields             uint32
	DmPositionX          int32
	DmPositionY          int32
	DmDisplayOrientation uint32
	DmDisplayFixedOutput uint32
	DmColor              int16
	DmDuplex             int16
	DmYResolution        int16
	DmTTOption           int16
	DmCollate            int16
	DmFormName           [32]uint16
	DmLogPixels          uint16
	DmBitsPerPel         uint32
	DmPelsWidth          uint32
	DmPelsHeight         uint32
	DmDisplayFlags       uint32
	DmDisplayFrequency   uint32
	DmICMMethod          uint32
	DmICMIntent          uint32
	DmMediaType          uint32
	DmDitherType         uint32
	DmReserved1          uint32
	DmReserved2          uint32
	DmPanningWidth       uint32
	DmPanningHeight      uint32
}

type DISPLAYCONFIG_DEVICE_INFO_HEADER struct {
	Type      uint32
	Size      uint32
	AdapterId windows.LUID
	Id        uint32
}

type DISPLAYCONFIG_SOURCE_DEVICE_NAME struct {
	Header            DISPLAYCONFIG_DEVICE_INFO_HEADER
	ViewGdiDeviceName [32]uint16
}

type DISPLAYCONFIG_GET_ADVANCED_COLOR_INFO struct {
	Header              DISPLAYCONFIG_DEVICE_INFO_HEADER
	Value               uint32
	ColorEncoding       uint32
	BitsPerColorChannel uint32
}

type DISPLAYCONFIG_SET_ADVANCED_COLOR_STATE struct {
	Header DISPLAYCONFIG_DEVICE_INFO_HEADER
	Value  uint32
}

type DISPLAYCONFIG_PATH_SOURCE_INFO struct {
	AdapterId   windows.LUID
	Id          uint32
	ModeInfoIdx uint32
	StatusFlags uint32
}

type DISPLAYCONFIG_RATIONAL struct {
	Numerator   uint32
	Denominator uint32
}

type DISPLAYCONFIG_PATH_TARGET_INFO struct {
	AdapterId        windows.LUID
	Id               uint32
	ModeInfoIdx      uint32
	OutputTechnology uint32
	Rotation         uint32
	Scaling          uint32
	RefreshRate      DISPLAYCONFIG_RATIONAL
	ScanLineOrdering uint32
	TargetAvailable  int32
	StatusFlags      uint32
}

type DISPLAYCONFIG_PATH_INFO struct {
	SourceInfo DISPLAYCONFIG_PATH_SOURCE_INFO
	TargetInfo DISPLAYCONFIG_PATH_TARGET_INFO
	Flags      uint32
}

type DISPLAYCONFIG_MODE_INFO struct {
	InfoType    uint32
	Id          uint32
	AdapterId   windows.LUID
	ModeInfoBuf [64]byte
}

type windowsManager struct {
	cache *Cache
	locks *IDLocks
	mu    sync.Mutex
	hdrMu sync.Mutex
	byID  map[string]monitorRef
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
	return &windowsManager{
		cache: NewCache(),
		locks: NewIDLocks(),
		byID:  make(map[string]monitorRef),
	}
}

func extractModelFromEDID(edid []byte) string {
	if len(edid) < 128 {
		return ""
	}
	offsets := []int{54, 72, 90, 108}
	for _, off := range offsets {
		if off+18 > len(edid) {
			break
		}
		desc := edid[off : off+18]
		if desc[0] == 0 && desc[1] == 0 && desc[2] == 0 && desc[3] == 0xFC {
			name := strings.TrimSpace(string(desc[5:]))
			name = strings.ReplaceAll(name, "\n", "")
			name = strings.ReplaceAll(name, "\r", "")
			name = strings.Trim(name, "\x00 ")
			if len(name) > 0 {
				return name
			}
		}
	}
	return ""
}

func queryEDIDModel(deviceID string) string {
	if deviceID == "" {
		return ""
	}
	if name := readEDIDModelAt(`SYSTEM\CurrentControlSet\Enum\` + deviceID + `\Device Parameters`); name != "" {
		return name
	}
	// MONITOR\SKG2409\{guid}\0002 → DISPLAY\SKG2409\<instance>\Device Parameters
	parts := strings.Split(deviceID, `\`)
	if len(parts) < 2 {
		return ""
	}
	vendor := parts[1]
	base := `SYSTEM\CurrentControlSet\Enum\DISPLAY\` + vendor
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, base, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return ""
	}
	defer k.Close()
	names, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return ""
	}
	for _, inst := range names {
		if name := readEDIDModelAt(base + `\` + inst + `\Device Parameters`); name != "" {
			return name
		}
	}
	return ""
}

func readEDIDModelAt(path string) string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	edid, _, err := k.GetBinaryValue("EDID")
	if err != nil {
		return ""
	}
	return extractModelFromEDID(edid)
}

func (m *windowsManager) GetMonitors(ctx context.Context) ([]Monitor, error) {
	if err := m.refreshTopology(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	ids := make([]string, 0, len(m.byID))
	for id := range m.byID {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	out := make([]Monitor, 0, len(ids))
	for _, id := range ids {
		m.mu.Lock()
		ref := m.byID[id]
		m.mu.Unlock()

		mon := Monitor{
			ID:         id,
			Name:       ref.name,
			DisplayNum: ref.displayNum,
			IsPrimary:  ref.isPrimary,
			Inputs:     DefaultInputPorts(),
		}

		if cached, ok := m.cache.Get(id); ok {
			mon.Brightness = cached.Brightness
			mon.CurrentPort = cached.CurrentPort
		}

		m.cache.Set(mon)
		out = append(out, mon)
	}
	return out, nil
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

func (m *windowsManager) SetBrightness(ctx context.Context, monitorID string, value uint32) error {
	err := m.setVCP(monitorID, VCPBrightness, value)
	if err == nil {
		if c, ok := m.cache.Get(monitorID); ok {
			c.Brightness = value
			m.cache.Set(c)
		}
	}
	return err
}

func (m *windowsManager) GetBrightness(ctx context.Context, monitorID string) (uint32, error) {
	if c, ok := m.cache.Get(monitorID); ok && c.Brightness > 0 {
		return c.Brightness, nil
	}
	return m.getVCP(monitorID, VCPBrightness)
}

func (m *windowsManager) SetInputSource(ctx context.Context, monitorID string, source uint32) error {
	err := m.setVCP(monitorID, VCPInput, source)
	if err == nil {
		if c, ok := m.cache.Get(monitorID); ok {
			c.CurrentPort = source
			m.cache.Set(c)
		}
	}
	return err
}

func (m *windowsManager) GetInputSource(ctx context.Context, monitorID string) (uint32, error) {
	return m.getVCP(monitorID, VCPInput)
}

func (m *windowsManager) SetVCP(ctx context.Context, monitorID string, code byte, value uint32) error {
	return m.setVCP(monitorID, code, value)
}

func (m *windowsManager) GetVCP(ctx context.Context, monitorID string, code byte) (uint32, uint32, error) {
	ref, err := m.resolve(monitorID)
	if err != nil {
		return 0, 0, err
	}
	unlock := m.locks.Lock(monitorID)
	defer unlock()
	return m.getVCPFeatureLocked(ref.hmon, code)
}

func (m *windowsManager) setVCP(monitorID string, code byte, value uint32) error {
	ref, err := m.resolve(monitorID)
	if err != nil {
		return err
	}
	unlock := m.locks.Lock(monitorID)
	defer unlock()
	return m.setVCPFeatureLocked(ref.hmon, code, value)
}

func (m *windowsManager) getVCP(monitorID string, code byte) (uint32, error) {
	ref, err := m.resolve(monitorID)
	if err != nil {
		return 0, err
	}
	unlock := m.locks.Lock(monitorID)
	defer unlock()
	return m.getVCPLocked(ref.hmon, code)
}

func (m *windowsManager) ListRefreshRates(ctx context.Context, monitorID string) ([]RefreshRate, error) {
	ref, err := m.resolve(monitorID)
	if err != nil {
		return nil, err
	}
	device := windows.StringToUTF16Ptr(ref.device)
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
		return nil, fmt.Errorf("display: no refresh rates found for %s", monitorID)
	}
	return out, nil
}

func (m *windowsManager) SetRefreshRate(ctx context.Context, monitorID string, rate RefreshRate) error {
	ref, err := m.resolve(monitorID)
	if err != nil {
		return err
	}
	device := windows.StringToUTF16Ptr(ref.device)
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

func (m *windowsManager) GetHDR(ctx context.Context, monitorID string) (bool, error) {
	ref, err := m.resolve(monitorID)
	if err != nil {
		return false, err
	}
	if !ref.hasHDR {
		return false, fmt.Errorf("display: no DisplayConfig targetId found for monitor %s", monitorID)
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

func (m *windowsManager) SetHDR(ctx context.Context, monitorID string, enabled bool) error {
	ref, err := m.resolve(monitorID)
	if err != nil {
		return err
	}
	if !ref.hasHDR {
		return fmt.Errorf("display: no DisplayConfig targetId found for monitor %s", monitorID)
	}

	m.hdrMu.Lock()
	defer m.hdrMu.Unlock()

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

func (m *windowsManager) Sleep(ctx context.Context, monitorIDs []string) error {
	if len(monitorIDs) == 0 {
		return ErrNoMonitorsSelected
	}

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

	for _, id := range monitorIDs {
		_ = m.setVCP(id, VCPPowerMode, 0x04)
	}
	return nil
}

func (m *windowsManager) Wake(ctx context.Context, monitorIDs []string) error {
	for _, id := range monitorIDs {
		_ = m.setVCP(id, VCPPowerMode, 0x01)
	}
	return sendWakeInput()
}

func sendWakeInput() error {
	procMouseEvent.Call(1, 0, 0, 0, 0)
	return nil
}

func (m *windowsManager) SubscribeChanges(ctx context.Context) (<-chan struct{}, error) {
	ch := make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(6 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = m.refreshTopology()
			}
		}
	}()
	return ch, nil
}

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

	const monitorInfoFPrimary = 0x00000001

	type pair struct {
		hmon      windows.Handle
		dev       string
		isPrimary bool
		rect      windows.Rect
	}
	var found []pair
	cb := syscall.NewCallback(func(hmon windows.Handle, hdc windows.Handle, rect *windows.Rect, lparam uintptr) uintptr {
		var info MONITORINFOEX
		info.CbSize = uint32(unsafe.Sizeof(info))
		r, _, _ := procGetMonitorInfoW.Call(uintptr(hmon), uintptr(unsafe.Pointer(&info)))
		if r == 0 {
			return 1
		}
		dev := windows.UTF16ToString(info.SzDevice[:])
		isPrimary := (info.DwFlags & monitorInfoFPrimary) != 0
		found = append(found, pair{
			hmon:      hmon,
			dev:       dev,
			isPrimary: isPrimary,
			rect:      info.RcMonitor,
		})
		return 1
	})
	r, _, _ = procEnumDisplayMonitors.Call(0, 0, cb, 0)

	next := make(map[string]monitorRef)
	for idx, p := range found {
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

func (m *windowsManager) getVCPLocked(hmon windows.Handle, code byte) (uint32, error) {
	cur, _, err := m.getVCPFeatureLocked(hmon, code)
	return cur, err
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
