//go:build windows

package mihomo

import "os/exec"

// KillAll stops stray mihomo.exe processes (port 2181/9090 leaks from failed connect).
func KillAll() {
	_ = exec.Command("taskkill", "/F", "/IM", "mihomo.exe").Run()
}
