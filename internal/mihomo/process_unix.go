//go:build !windows

package mihomo

import (
	"os/exec"
	"strings"
)

// KillAll stops stray mihomo child processes (zombies after failed connect/pretest).
func KillAll() {
	_ = exec.Command("pkill", "-x", "mihomo").Run()
	name := strings.TrimSuffix(strings.ToLower(ResolveBin()), ".exe")
	if name != "" && name != "mihomo" {
		_ = exec.Command("pkill", "-x", name).Run()
	}
}
