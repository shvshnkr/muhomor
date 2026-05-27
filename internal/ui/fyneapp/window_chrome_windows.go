//go:build windows && cgo

package fyneapp

import (
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

const (
	dwmwaUseImmersiveDarkMode   = 20
	dwmwaWindowCornerPreference = 33
	dwmwaBorderColor            = 34
	dwmwcpRound                 = 2
	dwmColorNone                = 0xFFFFFFFE
)

const windowCornerRadius = 12

func applyWindowShape(w fyne.Window) {
	nw, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	nw.RunNative(func(ctx any) {
		wc, ok := ctx.(driver.WindowsWindowContext)
		if !ok || wc.HWND == 0 {
			return
		}
		hwnd := wc.HWND
		applyDWMChrome(hwnd)
		if nativeIsZoomedHWND(hwnd) {
			clearWindowRegion(hwnd)
			return
		}
		applyWindowRegion(hwnd)
	})
}

func nativeIsZoomedHWND(hwnd uintptr) bool {
	user32 := syscall.NewLazyDLL("user32.dll")
	isZoomed := user32.NewProc("IsZoomed")
	r, _, _ := isZoomed.Call(hwnd)
	return r != 0
}

func clearWindowRegion(hwnd uintptr) {
	user32 := syscall.NewLazyDLL("user32.dll")
	setWindowRgn := user32.NewProc("SetWindowRgn")
	setWindowRgn.Call(hwnd, 0, 1)
}

func applyDWMChrome(hwnd uintptr) {
	dwm := syscall.NewLazyDLL("dwmapi.dll")
	setAttr := dwm.NewProc("DwmSetWindowAttribute")

	dark := int32(1)
	_, _, _ = setAttr.Call(hwnd, dwmwaUseImmersiveDarkMode, uintptr(unsafe.Pointer(&dark)), unsafe.Sizeof(dark))

	round := int32(dwmwcpRound)
	_, _, _ = setAttr.Call(hwnd, dwmwaWindowCornerPreference, uintptr(unsafe.Pointer(&round)), unsafe.Sizeof(round))

	border := uint32(dwmColorNone)
	_, _, _ = setAttr.Call(hwnd, dwmwaBorderColor, uintptr(unsafe.Pointer(&border)), unsafe.Sizeof(border))
}

func applyWindowRegion(hwnd uintptr) {
	user32 := syscall.NewLazyDLL("user32.dll")
	gdi32 := syscall.NewLazyDLL("gdi32.dll")
	getRect := user32.NewProc("GetWindowRect")
	createRoundRectRgn := gdi32.NewProc("CreateRoundRectRgn")
	setWindowRgn := user32.NewProc("SetWindowRgn")

	var rect struct {
		Left, Top, Right, Bottom int32
	}
	getRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	w := int(rect.Right - rect.Left)
	h := int(rect.Bottom - rect.Top)
	if w <= 0 || h <= 0 {
		return
	}
	r := windowCornerRadius
	rgn, _, _ := createRoundRectRgn.Call(
		0, 0,
		uintptr(w+1), uintptr(h+1),
		uintptr(r*2), uintptr(r*2),
	)
	if rgn == 0 {
		return
	}
	setWindowRgn.Call(hwnd, rgn, 1)
}
