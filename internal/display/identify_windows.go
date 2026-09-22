//go:build windows

package display

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	gdi32                          = windows.NewLazySystemDLL("gdi32.dll")
	procRegisterClassExW           = user32.NewProc("RegisterClassExW")
	procCreateWindowExW            = user32.NewProc("CreateWindowExW")
	procDefWindowProcW             = user32.NewProc("DefWindowProcW")
	procDestroyWindow              = user32.NewProc("DestroyWindow")
	procUnregisterClassW           = user32.NewProc("UnregisterClassW")
	procSetWindowLongPtrW          = user32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtrW          = user32.NewProc("GetWindowLongPtrW")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procShowWindow                 = user32.NewProc("ShowWindow")
	procUpdateWindow               = user32.NewProc("UpdateWindow")
	procBeginPaint                 = user32.NewProc("BeginPaint")
	procEndPaint                   = user32.NewProc("EndPaint")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessageW           = user32.NewProc("DispatchMessageW")
	procPostQuitMessage            = user32.NewProc("PostQuitMessage")
	procSetTimer                   = user32.NewProc("SetTimer")
	procKillTimer                  = user32.NewProc("KillTimer")
	procFillRect                   = user32.NewProc("FillRect")
	procDrawTextW                  = user32.NewProc("DrawTextW")

	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procCreateFontW      = gdi32.NewProc("CreateFontW")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procSetTextColor     = gdi32.NewProc("SetTextColor")
	procSetBkMode        = gdi32.NewProc("SetBkMode")
	procSelectObject     = gdi32.NewProc("SelectObject")
)

const (
	wsPopup          = 0x80000000
	wsVisible        = 0x10000000
	wsExTopMost      = 0x00000008
	wsExToolWindow   = 0x00000080
	wsExLayered      = 0x00080000
	wsExNoActivate   = 0x08000000
	swShowNoActivate = 4
	lwaAlpha         = 0x00000002
	wmPaint          = 0x000F
	wmTimer          = 0x0113
	wmDestroy        = 0x0002
	dtCenter         = 0x00000001
	dtVCenter        = 0x00000004
	dtSingleLine     = 0x00000020
	transparent  = 1
)

// GWLP_USERDATA == -21 as uintptr (two's complement).
var gwlpUserData = ^uintptr(20)

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type PAINTSTRUCT struct {
	Hdc         windows.Handle
	FErase      int32
	RcPaint     windows.Rect
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type MSG struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type monitorOverlayTarget struct {
	rect       windows.Rect
	displayNum int
}

func showIdentifyOverlays(targets []monitorOverlayTarget) {
	if len(targets) == 0 {
		return
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	className, _ := windows.UTF16PtrFromString("JustMonitorIdentifyOverlay")
	fontName, _ := windows.UTF16PtrFromString("Segoe UI")

	bgBrush, _, _ := procCreateSolidBrush.Call(uintptr(0x1F1F1F))
	font, _, _ := procCreateFontW.Call(
		84, 0, 0, 0, 700, 0, 0, 0, 1, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(fontName)),
	)
	defer procDeleteObject.Call(bgBrush)
	defer procDeleteObject.Call(font)

	wndProc := syscall.NewCallback(func(hwnd windows.Handle, msg uint32, wparam uintptr, lparam uintptr) uintptr {
		switch msg {
		case wmPaint:
			var ps PAINTSTRUCT
			hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
			num, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), gwlpUserData)

			procFillRect.Call(hdc, uintptr(unsafe.Pointer(&ps.RcPaint)), bgBrush)
			procSetBkMode.Call(hdc, uintptr(transparent))
			procSetTextColor.Call(hdc, uintptr(0xFFFFFF))
			oldFont, _, _ := procSelectObject.Call(hdc, font)

			textStr := fmt.Sprintf("%d", num)
			utf16Text, _ := windows.UTF16FromString(textStr)
			procDrawTextW.Call(
				hdc,
				uintptr(unsafe.Pointer(&utf16Text[0])),
				uintptr(len(utf16Text)-1),
				uintptr(unsafe.Pointer(&ps.RcPaint)),
				uintptr(dtCenter|dtVCenter|dtSingleLine),
			)

			procSelectObject.Call(hdc, oldFont)
			procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
			return 0

		case wmTimer:
			procKillTimer.Call(uintptr(hwnd), wparam)
			procPostQuitMessage.Call(0)
			return 0

		case wmDestroy:
			return 0
		}

		r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
		return r
	})

	var wc WNDCLASSEXW
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = wndProc
	wc.LpszClassName = className
	wc.HbrBackground = windows.Handle(bgBrush)
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	defer procUnregisterClassW.Call(uintptr(unsafe.Pointer(className)), 0)

	var windowsCreated []windows.Handle
	const cardW int32 = 140
	const cardH int32 = 140

	for _, t := range targets {
		cx := (t.rect.Left+t.rect.Right)/2 - (cardW / 2)
		cy := (t.rect.Top+t.rect.Bottom)/2 - (cardH / 2)

		hwnd, _, _ := procCreateWindowExW.Call(
			uintptr(wsExTopMost|wsExToolWindow|wsExLayered|wsExNoActivate),
			uintptr(unsafe.Pointer(className)),
			0,
			uintptr(wsPopup|wsVisible),
			uintptr(cx),
			uintptr(cy),
			uintptr(cardW),
			uintptr(cardH),
			0, 0, 0, 0,
		)
		if hwnd == 0 {
			continue
		}

		h := windows.Handle(hwnd)
		windowsCreated = append(windowsCreated, h)

		procSetWindowLongPtrW.Call(hwnd, gwlpUserData, uintptr(t.displayNum))
		procSetLayeredWindowAttributes.Call(hwnd, 0, 225, uintptr(lwaAlpha))
		procShowWindow.Call(hwnd, uintptr(swShowNoActivate))
		procUpdateWindow.Call(hwnd)
	}

	if len(windowsCreated) > 0 {
		procSetTimer.Call(uintptr(windowsCreated[0]), 1, 2500, 0)

		var msg MSG
		for {
			r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(r) <= 0 {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}

		for _, h := range windowsCreated {
			procDestroyWindow.Call(uintptr(h))
		}
	}
}
