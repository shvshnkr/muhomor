//go:build !windows

package platform

import "os/exec"

func startDaemonProcess(cmd *exec.Cmd) error {
	return cmd.Start()
}
