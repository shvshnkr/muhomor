//go:build !windows

package mihomo

import "os/exec"

func applyCmdAttrs(cmd *exec.Cmd) {}
