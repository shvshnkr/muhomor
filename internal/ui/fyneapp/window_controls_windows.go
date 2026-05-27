//go:build windows && cgo

package fyneapp

import (
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver"
)

const (
	swHide     = 0
	swMinimize = 6
	swMaximize = 3
	swRestore  = 9
)

func wireWindowControls(w fyne.Window, inner *container.InnerWindow) {
	if w == nil || inner == nil {
		return
	}
	inner.OnMinimized = func() {
		nativeShowWindow(w, swMinimize)
	}
	inner.OnMaximized = func() {
		if nativeIsZoomed(w) {
			nativeShowWindow(w, swRestore)
			inner.SetMaximized(false)
			syncWindowLayout(w, inner)
			return
		}
		nativeShowWindow(w, swMaximize)
		inner.SetMaximized(true)
		syncWindowLayout(w, inner)
	}
}

func nativeShowWindow(w fyne.Window, cmd int) {
	nw, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	nw.RunNative(func(ctx any) {
		wc, ok := ctx.(driver.WindowsWindowContext)
		if !ok || wc.HWND == 0 {
			return
		}
		user32 := syscall.NewLazyDLL("user32.dll")
		showWindow := user32.NewProc("ShowWindow")
		_, _, _ = showWindow.Call(wc.HWND, uintptr(cmd))
	})
}

func nativeIsZoomed(w fyne.Window) bool {
	zoomed := false
	nw, ok := w.(driver.NativeWindow)
	if !ok {
		return false
	}
	nw.RunNative(func(ctx any) {
		wc, ok := ctx.(driver.WindowsWindowContext)
		if !ok || wc.HWND == 0 {
			return
		}
		user32 := syscall.NewLazyDLL("user32.dll")
		isZoomed := user32.NewProc("IsZoomed")
		r, _, _ := isZoomed.Call(wc.HWND)
		zoomed = r != 0
	})
	return zoomed
}

func nativeHideWindow(w fyne.Window) {
	nativeShowWindow(w, swHide)
}
