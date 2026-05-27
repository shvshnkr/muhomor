//go:build windows

package platform

import (
	"syscall"
	"unsafe"
)

const (
	swRestore = 9
)

var (
	user32                   = syscall.NewLazyDLL("user32.dll")
	procFindWindowW          = user32.NewProc("FindWindowW")
	procShowWindow           = user32.NewProc("ShowWindow")
	procSetForegroundWindow  = user32.NewProc("SetForegroundWindow")
)

// ActivateGUIWindow shows an existing muhomor-gui window (same title as Wails options.App.Title).
func ActivateGUIWindow(title string) bool {
	if title == "" {
		title = "muhomor"
	}
	t, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return false
	}
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(t)))
	if hwnd == 0 {
		return false
	}
	_, _, _ = procShowWindow.Call(hwnd, uintptr(swRestore))
	_, _, _ = procSetForegroundWindow.Call(hwnd)
	return true
}
