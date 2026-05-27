//go:build windows

package platform

import (
	"syscall"
	"unsafe"
)

const mbOK = 0x00000000

var (
	procMessageBoxW = user32.NewProc("MessageBoxW")
)

// ShowErrorMessage shows a native error dialog (no Wails required).
func ShowErrorMessage(title, message string) {
	t, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	m, err := syscall.UTF16PtrFromString(message)
	if err != nil {
		return
	}
	_, _, _ = procMessageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), mbOK)
}
