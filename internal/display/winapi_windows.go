//go:build windows

package display

import (
	"time"

	"golang.org/x/sys/windows"
)

const (
	VCPPowerMode         = 0xD6
	powerModeOn          = 0x01
	powerModeOff         = 0x04
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
	procGetCapabilitiesStringLength             = dxva2.NewProc("GetCapabilitiesStringLength")
	procCapabilitiesRequestAndCapabilitiesReply = dxva2.NewProc("CapabilitiesRequestAndCapabilitiesReply")
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
