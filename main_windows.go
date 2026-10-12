//go:build windows

package main

// Windows kabuğu: yalnızca standart kütüphane + doğrudan Win32 çağrıları.
// Sahne yazılımla bir 32bpp DIB belleğine çizilir, yazılar GDI (ClearType) ile basılır.

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32   = syscall.NewLazyDLL("kernel32.dll")
	pGetSysDir = kernel32.NewProc("GetSystemDirectoryW")
	sysDir     = func() string {
		buf := make([]uint16, 300)
		n, _, _ := pGetSysDir.Call(uintptr(unsafe.Pointer(&buf[0])), 300)
		if n == 0 || n >= 300 {
			return `C:\Windows\System32`
		}
		return syscall.UTF16ToString(buf[:n])
	}()
	sysDLL = func(name string) *syscall.LazyDLL { return syscall.NewLazyDLL(filepath.Join(sysDir, name)) }

	user32   = sysDLL("user32.dll")
	gdi32    = sysDLL("gdi32.dll")
	shell32  = sysDLL("shell32.dll")
	winmm    = sysDLL("winmm.dll")
	dwmapi   = sysDLL("dwmapi.dll")
	advapi32 = sysDLL("advapi32.dll")
	ole32    = sysDLL("ole32.dll")

	pRegisterClassEx       = user32.NewProc("RegisterClassExW")
	pCreateWindowEx        = user32.NewProc("CreateWindowExW")
	pDefWindowProc         = user32.NewProc("DefWindowProcW")
	pDestroyWindow         = user32.NewProc("DestroyWindow")
	pShowWindow            = user32.NewProc("ShowWindow")
	pGetMessage            = user32.NewProc("GetMessageW")
	pTranslateMessage      = user32.NewProc("TranslateMessage")
	pDispatchMessage       = user32.NewProc("DispatchMessageW")
	pPostQuitMessage       = user32.NewProc("PostQuitMessage")
	pPostMessage           = user32.NewProc("PostMessageW")
	pSetTimer              = user32.NewProc("SetTimer")
	pInvalidateRect        = user32.NewProc("InvalidateRect")
	pBeginPaint            = user32.NewProc("BeginPaint")
	pEndPaint              = user32.NewProc("EndPaint")
	pGetClientRect         = user32.NewProc("GetClientRect")
	pGetWindowRect         = user32.NewProc("GetWindowRect")
	pLoadCursor            = user32.NewProc("LoadCursorW")
	pLoadImage             = user32.NewProc("LoadImageW")
	pSetCursor             = user32.NewProc("SetCursor")
	pSetWindowText         = user32.NewProc("SetWindowTextW")
	pSetWindowPos          = user32.NewProc("SetWindowPos")
	pSetWindowLongPtr      = user32.NewProc("SetWindowLongPtrW")
	pGetSystemMetrics      = user32.NewProc("GetSystemMetrics")
	pFlashWindowEx         = user32.NewProc("FlashWindowEx")
	pDrawText              = user32.NewProc("DrawTextW")
	pIsIconic              = user32.NewProc("IsIconic")
	pIsZoomed              = user32.NewProc("IsZoomed")
	pIsWindowVisible       = user32.NewProc("IsWindowVisible")
	pGetForegroundWindow   = user32.NewProc("GetForegroundWindow")
	pSetForegroundWindow   = user32.NewProc("SetForegroundWindow")
	pSetCapture            = user32.NewProc("SetCapture")
	pReleaseCapture        = user32.NewProc("ReleaseCapture")
	pTrackMouseEvent       = user32.NewProc("TrackMouseEvent")
	pGetDpiForWindow       = user32.NewProc("GetDpiForWindow")
	pSetDpiAwarenessCtx    = user32.NewProc("SetProcessDpiAwarenessContext")
	pSetProcessDPIAware    = user32.NewProc("SetProcessDPIAware")
	pMonitorFromWindow     = user32.NewProc("MonitorFromWindow")
	pMonitorFromRect       = user32.NewProc("MonitorFromRect")
	pGetMonitorInfo        = user32.NewProc("GetMonitorInfoW")
	pGetWindowPlacement    = user32.NewProc("GetWindowPlacement")
	pSetWindowPlacement    = user32.NewProc("SetWindowPlacement")
	pCreatePopupMenu       = user32.NewProc("CreatePopupMenu")
	pAppendMenu            = user32.NewProc("AppendMenuW")
	pTrackPopupMenu        = user32.NewProc("TrackPopupMenu")
	pDestroyMenu           = user32.NewProc("DestroyMenu")
	pGetCursorPos          = user32.NewProc("GetCursorPos")
	pScreenToClient        = user32.NewProc("ScreenToClient")
	pFindWindow            = user32.NewProc("FindWindowW")
	pOpenClipboard         = user32.NewProc("OpenClipboard")
	pCloseClipboard        = user32.NewProc("CloseClipboard")
	pGetClipboardData      = user32.NewProc("GetClipboardData")
	pRegisterWindowMessage = user32.NewProc("RegisterWindowMessageW")
	pGetKeyState           = user32.NewProc("GetKeyState")
	pSystemParametersInfo  = user32.NewProc("SystemParametersInfoW")
	pCreateIconIndirect    = user32.NewProc("CreateIconIndirect")

	pCreateCompatibleDC    = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection      = gdi32.NewProc("CreateDIBSection")
	pCreateBitmap          = gdi32.NewProc("CreateBitmap")
	pSelectObject          = gdi32.NewProc("SelectObject")
	pDeleteObject          = gdi32.NewProc("DeleteObject")
	pGetStockObject        = gdi32.NewProc("GetStockObject")
	pBitBlt                = gdi32.NewProc("BitBlt")
	pCreateFont            = gdi32.NewProc("CreateFontW")
	pSetBkMode             = gdi32.NewProc("SetBkMode")
	pSetTextColor          = gdi32.NewProc("SetTextColor")
	pSetTextCharacterExtra = gdi32.NewProc("SetTextCharacterExtra")
	pGetTextExtentPoint32  = gdi32.NewProc("GetTextExtentPoint32W")
	pGetTextFace           = gdi32.NewProc("GetTextFaceW")
	pGdiFlush              = gdi32.NewProc("GdiFlush")
	pSaveDC                = gdi32.NewProc("SaveDC")
	pRestoreDC             = gdi32.NewProc("RestoreDC")
	pIntersectClipRect     = gdi32.NewProc("IntersectClipRect")
	pGetDeviceCaps         = gdi32.NewProc("GetDeviceCaps")

	pGetModuleHandle = kernel32.NewProc("GetModuleHandleW")
	pCreateMutex     = kernel32.NewProc("CreateMutexW")
	pGlobalLock      = kernel32.NewProc("GlobalLock")
	pGlobalUnlock    = kernel32.NewProc("GlobalUnlock")

	pShellNotifyIcon       = shell32.NewProc("Shell_NotifyIconW")
	pPlaySound             = winmm.NewProc("PlaySoundW")
	pDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	pRegGetValue           = advapi32.NewProc("RegGetValueW")
	pCoInitializeEx        = ole32.NewProc("CoInitializeEx")
	pCoCreateInstance      = ole32.NewProc("CoCreateInstance")
)

const (
	wmDestroy           = 0x0002
	wmSize              = 0x0005
	wmPaint             = 0x000F
	wmClose             = 0x0010
	wmQueryEndSession   = 0x0011
	wmEraseBkgnd        = 0x0014
	wmSettingChange     = 0x001A
	wmSetCursor         = 0x0020
	wmGetMinMaxInfo     = 0x0024
	wmNcHitTest         = 0x0084
	wmNcLButtonDblClk   = 0x00A3
	wmKeyDown           = 0x0100
	wmChar              = 0x0102
	wmTimer             = 0x0113
	wmMouseMove         = 0x0200
	wmLButtonDown       = 0x0201
	wmLButtonUp         = 0x0202
	wmRButtonUp         = 0x0205
	wmMouseWheel        = 0x020A
	wmMouseLeave        = 0x02A3
	wmDpiChanged        = 0x02E0
	wmTray              = 0x8001
	ninBalloonUserClick = 0x0405

	wsOverlappedWindow = 0x00CF0000
	wsPopup            = 0x80000000
	wsVisible          = 0x10000000
	wsSysMenu          = 0x00080000
	wsMinimizeBox      = 0x00020000
	cwUseDefault       = 0x80000000

	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020
	swpShowWindow   = 0x0040

	swShowNormal    = 1
	swShowMinimized = 2
	swShowMaximized = 3
	swRestore       = 9

	srcCopy = 0x00CC0020

	dtLeft       = 0x0000
	dtCenter     = 0x0001
	dtRight      = 0x0002
	dtVCenter    = 0x0004
	dtSingleLine = 0x0020
	dtNoClip     = 0x0100
	dtNoPrefix   = 0x0800
)

var (
	hwndTopmost   = ^uintptr(0)  // -1
	hwndNoTopmost = ^uintptr(1)  // -2
	gwlStyle      = ^uintptr(15) // -16
)

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type point struct{ x, y int32 }

type winMsg struct {
	hwnd     uintptr
	message  uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	pt       point
	lPrivate uint32
}

type rect struct{ left, top, right, bottom int32 }

type paintStruct struct {
	hdc         uintptr
	fErase      int32
	rcPaint     rect
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

type bitmapInfo struct {
	size         uint32
	width        int32
	height       int32
	planes       uint16
	bitCount     uint16
	compression  uint32
	sizeImage    uint32
	xppm, yppm   int32
	clrUsed      uint32
	clrImportant uint32
	colors       [1]uint32
}

type flashInfo struct {
	cbSize  uint32
	hwnd    uintptr
	flags   uint32
	count   uint32
	timeout uint32
}

type trackMouse struct {
	cbSize    uint32
	dwFlags   uint32
	hwndTrack uintptr
	hoverTime uint32
}

type windowPlacement struct {
	length           uint32
	flags            uint32
	showCmd          uint32
	ptMinPosition    point
	ptMaxPosition    point
	rcNormalPosition rect
}

type monitorInfo struct {
	cbSize    uint32
	rcMonitor rect
	rcWork    rect
	dwFlags   uint32
}

type notifyIconData struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uVersion         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     uintptr
}

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  uintptr
	hbmColor uintptr
}

type guid struct {
	d1 uint32
	d2 uint16
	d3 uint16
	d4 [8]byte
}

// ---------------------------------------------------------------------------

type winHost struct {
	hwnd     uintptr
	hInst    uintptr
	memDC    uintptr
	dib      uintptr
	dibW     int
	dibH     int
	dpi      int
	fonts    map[Font]uintptr
	faces    map[string]bool
	wav      [2][]byte
	tbl      uintptr
	trayOK   bool
	iconBig  uintptr
	iconSm   uintptr
	compact  bool
	savedWP  windowPlacement
	tracking bool
	curs     [3]uintptr
	title    string
	tip      string
	chrome   *[3]uint32
	inChrome bool
	cfgDir   string
}

var (
	app           *App
	host          *winHost
	className     = "WorkXPomodoro"
	msgTaskbarNew uintptr
	msgTaskbarBtn uintptr
)

func init() { runtime.LockOSThread() }

func u16(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(strings.ReplaceAll(s, "\x00", ""))
	if err != nil {
		p, _ = syscall.UTF16PtrFromString("")
	}
	return p
}

func colorref(c RGB) uint32 {
	r, g, b := c.i()
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}

func copyU16(dst []uint16, s string) {
	src, _ := syscall.UTF16FromString(strings.ReplaceAll(s, "\x00", ""))
	if len(src) > len(dst) {
		src = src[:len(dst)]
		src[len(src)-1] = 0
	}
	copy(dst, src)
}

func logCrash(r any) {
	if host == nil || host.cfgDir == "" {
		return
	}
	_ = os.MkdirAll(host.cfgDir, 0o755)
	f, err := os.OpenFile(filepath.Join(host.cfgDir, "hata.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	fmt.Fprintf(f, "%s: %v\n%s\n", time.Now().Format(time.RFC3339), r, debug.Stack())
	f.Close()
}

func main() {
	cfgBase, err := os.UserConfigDir()
	if err != nil {
		cfgBase = os.TempDir()
	}
	host = &winHost{fonts: map[Font]uintptr{}, faces: map[string]bool{}, dpi: 96,
		cfgDir: filepath.Join(cfgBase, "Pomodoro")}
	defer func() {
		if r := recover(); r != nil {
			logCrash(r)
		}
	}()

	// DPI: manifest PerMonitorV2 bildiriyor; manifest yoksa da çalışsın diye çağır.
	if pSetDpiAwarenessCtx.Find() == nil {
		pSetDpiAwarenessCtx.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
	} else {
		pSetProcessDPIAware.Call()
	}

	// Tek örnek: ikinci kez açılırsa mevcut pencereyi öne getir.
	_, _, mErr := pCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(u16("Local\\WorkXPomodoro.Tek"))))
	if errno, ok := mErr.(syscall.Errno); ok && errno == 183 {
		if w, _, _ := pFindWindow.Call(uintptr(unsafe.Pointer(u16(className))), 0); w != 0 {
			if ic, _, _ := pIsIconic.Call(w); ic != 0 {
				pShowWindow.Call(w, swRestore)
			}
			pSetForegroundWindow.Call(w)
		}
		return
	}

	pCoInitializeEx.Call(0, 2)
	host.hInst, _, _ = pGetModuleHandle.Call(0)
	host.memDC, _, _ = pCreateCompatibleDC.Call(0)
	for i, id := range []uintptr{32512, 32649, 32513} {
		host.curs[i], _, _ = pLoadCursor.Call(0, id)
	}
	host.loadIcons()

	app = NewApp(host, filepath.Join(host.cfgDir, "ayarlar.json"))

	wc := wndClassEx{
		style:         0x0003, // CS_HREDRAW | CS_VREDRAW
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     host.hInst,
		hIcon:         host.iconBig,
		hIconSm:       host.iconSm,
		hCursor:       host.curs[0],
		lpszClassName: u16(className),
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	if atom, _, _ := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		return
	}
	msgTaskbarNew, _, _ = pRegisterWindowMessage.Call(uintptr(unsafe.Pointer(u16("TaskbarCreated"))))
	msgTaskbarBtn, _, _ = pRegisterWindowMessage.Call(uintptr(unsafe.Pointer(u16("TaskbarButtonCreated"))))

	hwnd, _, _ := pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(u16(className))), uintptr(unsafe.Pointer(u16("Pomodoro"))),
		wsOverlappedWindow, cwUseDefault, cwUseDefault, 1100, 760, 0, 0, host.hInst, 0)
	if hwnd == 0 {
		return
	}
	host.hwnd = hwnd
	host.dpi = host.getDpi()
	host.trayAdd()
	app.Update(time.Now())
	host.placeWindow()
	if app.S.Topmost {
		host.SetTopmost(true)
	}
	if app.S.StartMini {
		app.SetCompactMode(true)
	}
	pSetTimer.Call(hwnd, 1, 15, 0)

	var m winMsg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func lo16(v uintptr) int { return int(int16(v & 0xFFFF)) }
func hi16(v uintptr) int { return int(int16((v >> 16) & 0xFFFF)) }

func wndProc(hwnd, msg, wp, lp uintptr) (ret uintptr) {
	defer func() {
		if r := recover(); r != nil {
			logCrash(r)
			ret = 0
		}
	}()
	if host.hwnd == 0 {
		host.hwnd = hwnd
	}
	switch msg {
	case wmTimer:
		tick()
		return 0
	case wmPaint:
		host.paint()
		return 0
	case wmEraseBkgnd:
		return 1
	case wmSize:
		if wp != 1 { // SIZE_MINIMIZED değil
			host.resize(lo16(lp)&0xFFFF, hi16(lp)&0xFFFF)
		}
		return 0
	case wmGetMinMaxInfo:
		mm := (*[10]int32)(unsafe.Pointer(lp))
		sc := float64(host.getDpiFor(hwnd)) / 96
		if host.compact {
			mm[6], mm[7] = int32(220*sc), int32(72*sc)
		} else {
			mm[6], mm[7] = int32(780*sc), int32(600*sc)
		}
		return 0
	case wmDpiChanged:
		host.dpi = lo16(wp) & 0xFFFF
		r := (*rect)(unsafe.Pointer(lp))
		pSetWindowPos.Call(hwnd, 0, uintptr(r.left), uintptr(r.top), uintptr(r.right-r.left), uintptr(r.bottom-r.top), swpNoZOrder|swpNoActivate)
		var rc rect
		pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
		host.dibW = 0
		host.resize(int(rc.right), int(rc.bottom))
		return 0
	case wmSettingChange:
		if lp != 0 {
			s := syscall.UTF16ToString((*[64]uint16)(unsafe.Pointer(lp))[:])
			if s == "ImmersiveColorSet" {
				app.SystemThemeChanged(host.SystemDark())
			}
		}
	case wmSetCursor:
		if lo16(lp)&0xFFFF == 1 { // HTCLIENT
			pSetCursor.Call(host.curs[app.Cursor()])
			return 1
		}
	case wmNcHitTest:
		r, _, _ := pDefWindowProc.Call(hwnd, msg, wp, lp)
		if host.compact && r == 1 {
			p := point{int32(lo16(lp)), int32(hi16(lp))}
			pScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&p)))
			if app.Draggable(int(p.x), int(p.y)) {
				return 2 // HTCAPTION: boş alandan sürükle
			}
		}
		return r
	case wmNcLButtonDblClk:
		if host.compact {
			app.SetCompactMode(false)
			return 0
		}
	case wmMouseMove:
		if !host.tracking {
			tm := trackMouse{dwFlags: 2, hwndTrack: hwnd} // TME_LEAVE
			tm.cbSize = uint32(unsafe.Sizeof(tm))
			pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tm)))
			host.tracking = true
		}
		app.MouseMove(lo16(lp), hi16(lp))
		return 0
	case wmMouseLeave:
		host.tracking = false
		app.MouseLeave()
		return 0
	case wmLButtonDown:
		pSetCapture.Call(hwnd)
		app.MouseDown(lo16(lp), hi16(lp))
		return 0
	case wmLButtonUp:
		pReleaseCapture.Call()
		app.MouseUp(lo16(lp), hi16(lp))
		return 0
	case wmMouseWheel:
		p := point{int32(lo16(lp)), int32(hi16(lp))}
		pScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&p)))
		app.Wheel(hi16(wp), int(p.x), int(p.y))
		return 0
	case wmKeyDown:
		vk := int(wp)
		repeat := lp&(1<<30) != 0
		ks, _, _ := pGetKeyState.Call(0x11)
		ctrl := ks&0x8000 != 0
		if repeat && !app.editing {
			switch vk {
			case 0x26, 0x28, 0x6B, 0x6D, 0xBB, 0xBD:
			default:
				return 0
			}
		}
		app.KeyDown(vk, ctrl, false)
		return 0
	case wmChar:
		app.Char(rune(wp))
		return 0
	case wmTray:
		switch lo16(lp) & 0xFFFF {
		case wmLButtonUp, ninBalloonUserClick:
			host.restore()
		case wmRButtonUp:
			host.trayMenu()
		}
		return 0
	case wmQueryEndSession:
		host.savePlacement()
		app.SaveNow()
		return 1
	case wmClose:
		host.savePlacement()
		pDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		host.trayDelete()
		app.SaveNow()
		pPostQuitMessage.Call(0)
		return 0
	}
	if msgTaskbarNew != 0 && msg == msgTaskbarNew {
		host.trayAdd()
		return 0
	}
	if msgTaskbarBtn != 0 && msg == msgTaskbarBtn {
		host.taskbarInit()
	}
	r, _, _ := pDefWindowProc.Call(hwnd, msg, wp, lp)
	return r
}

func tick() {
	app.Update(time.Now())
	if ic, _, _ := pIsIconic.Call(host.hwnd); ic != 0 || host.dibW == 0 {
		return
	}
	if app.WantsFrame() {
		app.Render()
		pInvalidateRect.Call(host.hwnd, 0, 0)
	}
}

// ---------------------------------------------------------------------------
// Pencere, DIB ve DPI
// ---------------------------------------------------------------------------

func (h *winHost) getDpiFor(hwnd uintptr) int {
	if hwnd != 0 && pGetDpiForWindow.Find() == nil {
		if d, _, _ := pGetDpiForWindow.Call(hwnd); d > 0 {
			return int(d)
		}
	}
	d, _, _ := pGetDeviceCaps.Call(h.memDC, 88) // LOGPIXELSX
	if d == 0 {
		return 96
	}
	return int(d)
}

func (h *winHost) getDpi() int { return h.getDpiFor(h.hwnd) }

func (h *winHost) resize(w, hh int) {
	if w <= 0 || hh <= 0 || h.memDC == 0 {
		return
	}
	if w != h.dibW || hh != h.dibH {
		bi := bitmapInfo{size: 40, width: int32(w), height: -int32(hh), planes: 1, bitCount: 32}
		var bits unsafe.Pointer
		nb, _, _ := pCreateDIBSection.Call(h.memDC, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
		if nb == 0 || bits == nil {
			return
		}
		pSelectObject.Call(h.memDC, nb)
		old := h.dib
		h.dib, h.dibW, h.dibH = nb, w, hh
		app.Resize(w, hh, unsafe.Slice((*uint32)(bits), w*hh), float64(h.dpi)/96)
		if old != 0 {
			pDeleteObject.Call(old)
		}
	} else {
		app.Resize(w, hh, app.c.Pix, float64(h.dpi)/96)
	}
	app.Render()
	pInvalidateRect.Call(h.hwnd, 0, 0)
}

func (h *winHost) paint() {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(h.hwnd, uintptr(unsafe.Pointer(&ps)))
	if h.dib != 0 {
		pBitBlt.Call(hdc, 0, 0, uintptr(h.dibW), uintptr(h.dibH), h.memDC, 0, 0, srcCopy)
	}
	pEndPaint.Call(h.hwnd, uintptr(unsafe.Pointer(&ps)))
}

func (h *winHost) workArea() rect {
	mon, _, _ := pMonitorFromWindow.Call(h.hwnd, 2)
	mi := monitorInfo{}
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	if mon != 0 {
		if r, _, _ := pGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi))); r != 0 {
			return mi.rcWork
		}
	}
	w, _, _ := pGetSystemMetrics.Call(0)
	hh, _, _ := pGetSystemMetrics.Call(1)
	return rect{0, 0, int32(w), int32(hh)}
}

func onScreen(r rect) bool {
	m, _, _ := pMonitorFromRect.Call(uintptr(unsafe.Pointer(&r)), 0)
	return m != 0
}

func (h *winHost) placeWindow() {
	s := app.S.Win
	if s.Ok && s.W >= 300 && s.H >= 200 {
		rc := rect{int32(s.X), int32(s.Y), int32(s.X + s.W), int32(s.Y + s.H)}
		if onScreen(rc) {
			wp := windowPlacement{showCmd: swShowNormal, rcNormalPosition: rc}
			if s.Max {
				wp.showCmd = swShowMaximized
			}
			wp.length = uint32(unsafe.Sizeof(wp))
			if r, _, _ := pSetWindowPlacement.Call(h.hwnd, uintptr(unsafe.Pointer(&wp))); r != 0 {
				return
			}
		}
	}
	sc := float64(h.getDpi()) / 96
	wa := h.workArea()
	aw, ah := float64(wa.right-wa.left), float64(wa.bottom-wa.top)
	w := min(1120*sc, aw*0.92)
	hh := min(800*sc, ah*0.92)
	x := float64(wa.left) + (aw-w)/2
	y := float64(wa.top) + (ah-hh)/2
	pSetWindowPos.Call(h.hwnd, 0, uintptr(int32(x)), uintptr(int32(y)), uintptr(int32(w)), uintptr(int32(hh)), swpNoZOrder|swpNoActivate)
	pShowWindow.Call(h.hwnd, swShowNormal)
}

func (h *winHost) savePlacement() {
	if h.hwnd == 0 {
		return
	}
	store := func(wp windowPlacement) {
		rc := wp.rcNormalPosition
		app.S.Win = Place{X: int(rc.left), Y: int(rc.top), W: int(rc.right - rc.left), H: int(rc.bottom - rc.top),
			Max: wp.showCmd == swShowMaximized || (wp.showCmd == swShowMinimized && wp.flags&2 != 0), Ok: true}
	}
	if h.compact {
		var r rect
		pGetWindowRect.Call(h.hwnd, uintptr(unsafe.Pointer(&r)))
		app.S.Mini = Place{X: int(r.left), Y: int(r.top), Ok: true}
		if h.savedWP.length != 0 {
			store(h.savedWP)
		}
		return
	}
	wp := windowPlacement{}
	wp.length = uint32(unsafe.Sizeof(wp))
	if r, _, _ := pGetWindowPlacement.Call(h.hwnd, uintptr(unsafe.Pointer(&wp))); r != 0 {
		store(wp)
	}
}

func (h *winHost) restore() {
	if ic, _, _ := pIsIconic.Call(h.hwnd); ic != 0 {
		pShowWindow.Call(h.hwnd, swRestore)
	}
	pSetForegroundWindow.Call(h.hwnd)
}

func (h *winHost) dwmSet(attr uintptr, v uint32) bool {
	if pDwmSetWindowAttribute.Find() != nil || h.hwnd == 0 {
		return false
	}
	r, _, _ := pDwmSetWindowAttribute.Call(h.hwnd, attr, uintptr(unsafe.Pointer(&v)), 4)
	return r == 0
}

// ---------------------------------------------------------------------------
// Host arayüzü
// ---------------------------------------------------------------------------

func (h *winHost) faceOK(name string) bool {
	if v, ok := h.faces[name]; ok {
		return v
	}
	hf, _, _ := pCreateFont.Call(uintptr(^uint32(11)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(u16(name))))
	ok := false
	if hf != 0 {
		old, _, _ := pSelectObject.Call(h.memDC, hf)
		buf := make([]uint16, 64)
		n, _, _ := pGetTextFace.Call(h.memDC, 64, uintptr(unsafe.Pointer(&buf[0])))
		pSelectObject.Call(h.memDC, old)
		pDeleteObject.Call(hf)
		if n > 0 && int(n) <= 64 {
			ok = strings.EqualFold(syscall.UTF16ToString(buf[:n]), name)
		}
	}
	h.faces[name] = ok
	return ok
}

func (h *winHost) pick(names ...string) string {
	for _, n := range names[:len(names)-1] {
		if h.faceOK(n) {
			return n
		}
	}
	return names[len(names)-1]
}

func (h *winHost) font(f Font) uintptr {
	if hf, ok := h.fonts[f]; ok {
		return hf
	}
	var name string
	var weight int
	switch {
	case f.Face == faceDisp && f.Weight <= 350:
		name, weight = h.pick("Segoe UI Variable Display Light", "Segoe UI Light"), 300
	case f.Face == faceDisp && f.Weight >= 550:
		name, weight = h.pick("Segoe UI Variable Display Semibold", "Segoe UI Semibold"), 600
	case f.Face == faceDisp:
		name, weight = h.pick("Segoe UI Variable Display", "Segoe UI"), 400
	case f.Weight >= 650:
		name, weight = h.pick("Segoe UI Variable Text", "Segoe UI"), 700
	case f.Weight >= 550:
		name, weight = h.pick("Segoe UI Variable Text Semibold", "Segoe UI Semibold"), 600
	default:
		name, weight = h.pick("Segoe UI Variable Text", "Segoe UI"), 400
	}
	size := f.Size
	if size < 1 {
		size = 1
	}
	hf, _, _ := pCreateFont.Call(uintptr(uint32(int32(-size))), 0, 0, 0, uintptr(weight), 0, 0, 0,
		1 /*DEFAULT_CHARSET*/, 4 /*OUT_TT_PRECIS*/, 0, 5 /*CLEARTYPE_QUALITY*/, 0, uintptr(unsafe.Pointer(u16(name))))
	h.fonts[f] = hf
	return hf
}

func (h *winHost) clearFonts() {
	stock, _, _ := pGetStockObject.Call(17) // DEFAULT_GUI_FONT
	pSelectObject.Call(h.memDC, stock)
	for k, v := range h.fonts {
		pDeleteObject.Call(v)
		delete(h.fonts, k)
	}
}

func (h *winHost) Measure(f Font, s string) int {
	if s == "" || h.memDC == 0 {
		return 0
	}
	old, _, _ := pSelectObject.Call(h.memDC, h.font(f))
	pSetTextCharacterExtra.Call(h.memDC, uintptr(f.Spacing))
	u, err := syscall.UTF16FromString(s)
	if err != nil || len(u) < 2 {
		return 0
	}
	var sz [2]int32
	pGetTextExtentPoint32.Call(h.memDC, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&sz)))
	pSelectObject.Call(h.memDC, old)
	return int(sz[0])
}

func (h *winHost) DrawText(cmds []TextCmd) {
	if h.memDC == 0 || h.dib == 0 || len(cmds) == 0 {
		return
	}
	if len(h.fonts) > 64 {
		h.clearFonts()
	}
	dc := h.memDC
	pSetBkMode.Call(dc, 1) // TRANSPARENT
	old, _, _ := pSelectObject.Call(dc, h.font(cmds[0].F))
	for _, t := range cmds {
		u, err := syscall.UTF16FromString(t.S)
		if err != nil || len(u) < 2 {
			continue
		}
		pSelectObject.Call(dc, h.font(t.F))
		pSetTextColor.Call(dc, uintptr(colorref(t.C)))
		pSetTextCharacterExtra.Call(dc, uintptr(t.F.Spacing))
		x, y := int32(t.X), int32(t.Y)
		flags := uintptr(dtSingleLine | dtVCenter | dtNoPrefix | dtNoClip)
		var rc rect
		switch t.Align {
		case alignL:
			rc = rect{x, y - 400, x + 8000, y + 400}
			flags |= dtLeft
		case alignR:
			rc = rect{x - 8000, y - 400, x, y + 400}
			flags |= dtRight
		default:
			rc = rect{x - 4000, y - 400, x + 4000, y + 400}
			flags |= dtCenter
		}
		if t.Clip {
			pSaveDC.Call(dc)
			pIntersectClipRect.Call(dc, uintptr(int32(t.CX0)), 0, uintptr(int32(t.CX1)), uintptr(int32(h.dibH)))
		}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&rc)), flags)
		if t.Clip {
			pRestoreDC.Call(dc, ^uintptr(0))
		}
	}
	pSelectObject.Call(dc, old)
	pGdiFlush.Call()
}

func (h *winHost) PlaySound(wav []byte) {
	if len(wav) == 0 || pPlaySound.Find() != nil {
		return
	}
	h.wav[1] = h.wav[0] // önceki tampon, çalma durdurulana kadar canlı kalsın
	h.wav[0] = wav
	pPlaySound.Call(uintptr(unsafe.Pointer(&wav[0])), 0, 0x0001|0x0002|0x0004) // SND_ASYNC|SND_NODEFAULT|SND_MEMORY
}

func (h *winHost) nid() notifyIconData {
	n := notifyIconData{hWnd: h.hwnd, uID: 1}
	n.cbSize = uint32(unsafe.Sizeof(n))
	return n
}

func (h *winHost) trayAdd() {
	if h.hwnd == 0 {
		return
	}
	n := h.nid()
	n.uFlags = 0x1 | 0x2 | 0x4 // NIF_MESSAGE | NIF_ICON | NIF_TIP
	n.uCallbackMessage = wmTray
	n.hIcon = h.iconSm
	tip := h.tip
	if tip == "" {
		tip = "Pomodoro"
	}
	copyU16(n.szTip[:], tip)
	r, _, _ := pShellNotifyIcon.Call(0, uintptr(unsafe.Pointer(&n)))
	h.trayOK = r != 0
}

func (h *winHost) trayDelete() {
	if !h.trayOK {
		return
	}
	n := h.nid()
	pShellNotifyIcon.Call(2, uintptr(unsafe.Pointer(&n)))
	h.trayOK = false
}

func (h *winHost) SetTrayTip(s string) {
	h.tip = s
	if !h.trayOK {
		return
	}
	n := h.nid()
	n.uFlags = 0x4
	copyU16(n.szTip[:], s)
	pShellNotifyIcon.Call(1, uintptr(unsafe.Pointer(&n)))
}

func (h *winHost) Notify(title, body string) {
	if !h.trayOK {
		return
	}
	n := h.nid()
	n.uFlags = 0x10 // NIF_INFO
	copyU16(n.szInfo[:], body)
	copyU16(n.szInfoTitle[:], title)
	n.dwInfoFlags = 0x4 | 0x20 // NIIF_USER | NIIF_LARGE_ICON
	n.hBalloonIcon = h.iconBig
	pShellNotifyIcon.Call(1, uintptr(unsafe.Pointer(&n)))
}

func (h *winHost) trayMenu() {
	m, _, _ := pCreatePopupMenu.Call()
	if m == 0 {
		return
	}
	add := func(id uintptr, s string) {
		pAppendMenu.Call(m, 0, id, uintptr(unsafe.Pointer(u16(s))))
	}
	sep := func() { pAppendMenu.Call(m, 0x800, 0, 0) }
	add(1, app.playTip())
	add(2, "Sıfırla")
	add(3, "Sonrakine geç")
	sep()
	if h.compact {
		add(4, "Tam pencere")
	} else {
		add(4, "Kompakt pencere")
	}
	add(5, "Göster")
	sep()
	add(6, "Çıkış")
	var p point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	pSetForegroundWindow.Call(h.hwnd)
	cmd, _, _ := pTrackPopupMenu.Call(m, 0x0100|0x0002|0x0020, uintptr(p.x), uintptr(p.y), 0, h.hwnd, 0)
	pPostMessage.Call(h.hwnd, 0, 0, 0)
	pDestroyMenu.Call(m)
	switch cmd {
	case 1:
		app.toggle()
	case 2:
		app.reset()
	case 3:
		app.skip()
	case 4:
		app.SetCompactMode(!h.compact)
	case 5:
		h.restore()
	case 6:
		h.savePlacement()
		pDestroyWindow.Call(h.hwnd)
	}
}

func (h *winHost) Attention() {
	if fg, _, _ := pGetForegroundWindow.Call(); fg == h.hwnd {
		return
	}
	fi := flashInfo{hwnd: h.hwnd, flags: 0x3 | 0xC, count: 3} // FLASHW_ALL | FLASHW_TIMERNOFG
	fi.cbSize = uint32(unsafe.Sizeof(fi))
	pFlashWindowEx.Call(uintptr(unsafe.Pointer(&fi)))
}

func (h *winHost) SetTopmost(on bool) {
	if h.hwnd == 0 || h.compact {
		return
	}
	after := hwndNoTopmost
	if on {
		after = hwndTopmost
	}
	pSetWindowPos.Call(h.hwnd, after, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
}

func (h *winHost) SetCompact(on bool) {
	if on == h.compact || h.hwnd == 0 {
		return
	}
	sc := float64(h.getDpi()) / 96
	if on {
		h.savedWP = windowPlacement{}
		h.savedWP.length = uint32(unsafe.Sizeof(h.savedWP))
		pGetWindowPlacement.Call(h.hwnd, uintptr(unsafe.Pointer(&h.savedWP)))
		if z, _, _ := pIsZoomed.Call(h.hwnd); z != 0 {
			pShowWindow.Call(h.hwnd, swRestore)
		}
		h.compact = true
		w, hh := int32(300*sc), int32(100*sc)
		wa := h.workArea()
		x, y := wa.right-w-int32(24*sc), wa.bottom-hh-int32(24*sc)
		if m := app.S.Mini; m.Ok && onScreen(rect{int32(m.X), int32(m.Y), int32(m.X) + w, int32(m.Y) + hh}) {
			x, y = int32(m.X), int32(m.Y)
		}
		pSetWindowLongPtr.Call(h.hwnd, gwlStyle, wsPopup|wsVisible|wsSysMenu|wsMinimizeBox)
		pSetWindowPos.Call(h.hwnd, hwndTopmost, uintptr(x), uintptr(y), uintptr(w), uintptr(hh), swpFrameChanged|swpShowWindow)
		h.dwmSet(33, 2) // DWMWA_WINDOW_CORNER_PREFERENCE = yuvarlak (Windows 11)
		return
	}
	var r rect
	pGetWindowRect.Call(h.hwnd, uintptr(unsafe.Pointer(&r)))
	app.S.Mini = Place{X: int(r.left), Y: int(r.top), Ok: true}
	h.compact = false
	pSetWindowLongPtr.Call(h.hwnd, gwlStyle, wsOverlappedWindow|wsVisible)
	h.dwmSet(33, 0)
	after := hwndNoTopmost
	if app.S.Topmost {
		after = hwndTopmost
	}
	pSetWindowPos.Call(h.hwnd, after, 0, 0, 0, 0, swpNoMove|swpNoSize|swpFrameChanged|swpShowWindow)
	if h.savedWP.length != 0 {
		wp := h.savedWP
		if wp.showCmd != swShowMaximized {
			wp.showCmd = swShowNormal
		}
		pSetWindowPlacement.Call(h.hwnd, uintptr(unsafe.Pointer(&wp)))
	}
	if h.chrome != nil {
		c := *h.chrome
		h.applyChrome(c[0] == 1, c[1], c[2])
	}
}

func (h *winHost) SystemDark() bool {
	var v, size uint32 = 1, 4
	r, _, _ := pRegGetValue.Call(0x80000001,
		uintptr(unsafe.Pointer(u16(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`))),
		uintptr(unsafe.Pointer(u16("AppsUseLightTheme"))), 0x10, 0, uintptr(unsafe.Pointer(&v)), uintptr(unsafe.Pointer(&size)))
	return r == 0 && v == 0
}

func (h *winHost) ReduceMotion() bool {
	var b int32 = 1
	if r, _, _ := pSystemParametersInfo.Call(0x1042, 0, uintptr(unsafe.Pointer(&b)), 0); r == 0 {
		return false
	}
	return b == 0
}

func (h *winHost) SetTitle(s string) {
	h.title = s
	if h.hwnd != 0 {
		pSetWindowText.Call(h.hwnd, uintptr(unsafe.Pointer(u16(s))))
	}
}

func (h *winHost) applyChrome(dark bool, capc, txt uint32) {
	v := uint32(0)
	if dark {
		v = 1
	}
	if !h.dwmSet(20, v) { // DWMWA_USE_IMMERSIVE_DARK_MODE
		h.dwmSet(19, v) // Windows 10 1809–1909
	}
	h.dwmSet(35, capc) // DWMWA_CAPTION_COLOR (Windows 11)
	h.dwmSet(36, txt)  // DWMWA_TEXT_COLOR
	// Windows 10 başlık çubuğunu hemen yeniden boyasın (yalnızca görünürken, iç içe çağrı olmadan)
	if vis, _, _ := pIsWindowVisible.Call(h.hwnd); vis != 0 && !h.inChrome {
		h.inChrome = true
		pSetWindowPos.Call(h.hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged)
		h.inChrome = false
	}
}

func (h *winHost) SetChrome(dark bool, capc, txt RGB) {
	d := uint32(0)
	if dark {
		d = 1
	}
	h.chrome = &[3]uint32{d, colorref(capc), colorref(txt)}
	if h.hwnd != 0 && !h.compact {
		h.applyChrome(dark, colorref(capc), colorref(txt))
	}
}

func (h *winHost) taskbarInit() {
	if h.tbl != 0 {
		app.lastProg = [2]int{-1, -1}
		return
	}
	clsid := guid{0x56FDF344, 0xFD6D, 0x11d0, [8]byte{0x95, 0x8A, 0x00, 0x60, 0x97, 0xC9, 0xA0, 0x90}}
	iid := guid{0xEA1AFB91, 0x9E28, 0x4B86, [8]byte{0x90, 0xE9, 0x9E, 0x9F, 0x8A, 0x5E, 0xEF, 0xAF}}
	var obj uintptr
	hr, _, _ := pCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsid)), 0, 1, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&obj)))
	if hr != 0 || obj == 0 {
		return
	}
	h.tbl = obj
	syscall.SyscallN(h.vtbl(3), obj) // HrInit
	app.lastProg = [2]int{-1, -1}    // durumu yeniden gönder
}

func (h *winHost) vtbl(i int) uintptr {
	vt := *(*uintptr)(unsafe.Pointer(h.tbl))
	return *(*uintptr)(unsafe.Pointer(vt + uintptr(i)*unsafe.Sizeof(uintptr(0))))
}

func (h *winHost) SetProgress(state int, frac float64) {
	if h.tbl == 0 || h.hwnd == 0 {
		return
	}
	flags := uintptr(0) // TBPF_NOPROGRESS
	switch state {
	case 1:
		flags = 2 // TBPF_NORMAL
	case 2:
		flags = 8 // TBPF_PAUSED
	}
	syscall.SyscallN(h.vtbl(10), h.tbl, h.hwnd, flags) // SetProgressState
	if state != 0 {
		v := uintptr(clampf(frac, 0, 1) * 1000)
		syscall.SyscallN(h.vtbl(9), h.tbl, h.hwnd, v, 1000) // SetProgressValue
	}
}

func (h *winHost) Clipboard() string {
	if r, _, _ := pOpenClipboard.Call(h.hwnd); r == 0 {
		return ""
	}
	defer pCloseClipboard.Call()
	hd, _, _ := pGetClipboardData.Call(13) // CF_UNICODETEXT
	if hd == 0 {
		return ""
	}
	p, _, _ := pGlobalLock.Call(hd)
	if p == 0 {
		return ""
	}
	defer pGlobalUnlock.Call(hd)
	buf := (*[4096]uint16)(unsafe.Pointer(p))
	n := 0
	for n < 4096 && buf[n] != 0 {
		n++
	}
	return syscall.UTF16ToString(buf[:n])
}

// ---------------------------------------------------------------------------
// Simgeler: gömülü kaynaktan (ID 1); yoksa çalışma anında çizilir.
// ---------------------------------------------------------------------------

func (h *winHost) loadIcons() {
	cx, _, _ := pGetSystemMetrics.Call(11)
	sx, _, _ := pGetSystemMetrics.Call(49)
	h.iconBig, _, _ = pLoadImage.Call(h.hInst, 1, 1, cx, cx, 0)
	h.iconSm, _, _ = pLoadImage.Call(h.hInst, 1, 1, sx, sx, 0)
	if h.iconBig == 0 {
		h.iconBig = makeIcon(int(cx))
	}
	if h.iconSm == 0 {
		h.iconSm = makeIcon(int(sx))
	}
}

func makeIcon(size int) uintptr {
	if size <= 0 {
		size = 32
	}
	c := NewCanvas(size, size)
	DrawAppIcon(c)
	bi := bitmapInfo{size: 40, width: int32(size), height: -int32(size), planes: 1, bitCount: 32}
	var bits unsafe.Pointer
	color, _, _ := pCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if color == 0 || bits == nil {
		return 0
	}
	copy(unsafe.Slice((*uint32)(bits), size*size), c.Pix)
	mask, _, _ := pCreateBitmap.Call(uintptr(size), uintptr(size), 1, 1, 0)
	ii := iconInfo{fIcon: 1, hbmMask: mask, hbmColor: color}
	ic, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	pDeleteObject.Call(color)
	pDeleteObject.Call(mask)
	return ic
}
