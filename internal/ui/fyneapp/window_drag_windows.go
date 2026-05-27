//go:build windows && cgo

package fyneapp

import (
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver"
)

func wireWindowDrag(w fyne.Window, inner *container.InnerWindow) {
	inner.OnDragged = func(ev *fyne.DragEvent) {
		moveWindowBy(w, ev.Dragged)
	}
}

func resizeWindowBy(w fyne.Window, delta fyne.Delta) {
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
		getRect := user32.NewProc("GetWindowRect")
		setPos := user32.NewProc("SetWindowPos")
		var rect struct {
			Left, Top, Right, Bottom int32
		}
		getRect.Call(wc.HWND, uintptr(unsafe.Pointer(&rect)))
		wPx := int(rect.Right - rect.Left)
		hPx := int(rect.Bottom - rect.Top)
		newW := wPx + int(delta.DX)
		newH := hPx + int(delta.DY)
		if newW < 320 {
			newW = 320
		}
		if newH < 240 {
			newH = 240
		}
		const swpNoMove = 0x0002
		setPos.Call(wc.HWND, 0, 0, 0, uintptr(newW), uintptr(newH), swpNoMove)
	})
}

func moveWindowBy(w fyne.Window, delta fyne.Delta) {
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
		getRect := user32.NewProc("GetWindowRect")
		setPos := user32.NewProc("SetWindowPos")
		var rect struct {
			Left, Top, Right, Bottom int32
		}
		getRect.Call(wc.HWND, uintptr(unsafe.Pointer(&rect)))
		x := int(rect.Left) + int(delta.DX)
		y := int(rect.Top) + int(delta.DY)
		const swpNoSize = 0x0001
		setPos.Call(wc.HWND, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize)
	})
}
