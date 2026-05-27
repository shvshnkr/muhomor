//go:build windows && cgo

package fyneapp

import (
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

func syncNativeWindowSize(w fyne.Window) fyne.Size {
	cur := w.Canvas().Size()
	nw, ok := w.(driver.NativeWindow)
	if !ok {
		return cur
	}
	var native fyne.Size
	var valid bool
	nw.RunNative(func(ctx any) {
		wc, ok := ctx.(driver.WindowsWindowContext)
		if !ok || wc.HWND == 0 {
			return
		}
		user32 := syscall.NewLazyDLL("user32.dll")
		getClientRect := user32.NewProc("GetClientRect")
		var rect struct {
			Left, Top, Right, Bottom int32
		}
		getClientRect.Call(wc.HWND, uintptr(unsafe.Pointer(&rect)))
		cw := float32(rect.Right - rect.Left)
		ch := float32(rect.Bottom - rect.Top)
		if cw <= 0 || ch <= 0 {
			return
		}
		scale := w.Canvas().Scale()
		if scale <= 0 {
			scale = 1
		}
		native = fyne.NewSize(cw/scale, ch/scale)
		valid = true
	})
	if !valid {
		return cur
	}
	if cur != native {
		w.Resize(native)
	}
	return native
}
