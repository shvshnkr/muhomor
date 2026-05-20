//go:build !windows

package platform

import "os/exec"

func OpenPath(path string) error {
	return exec.Command("xdg-open", path).Start()
}
