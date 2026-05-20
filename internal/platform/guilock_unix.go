//go:build !windows

package platform

import "syscall"

func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}
