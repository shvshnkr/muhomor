//go:build windows

package platform

import "syscall"

const processQueryLimited = 0x1000

func processAlive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimited, false, uint32(pid))
	if err != nil {
		return false
	}
	_ = syscall.CloseHandle(h)
	return true
}
