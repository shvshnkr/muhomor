//go:build windows

package platform

import "os/exec"

// OpenPath opens file or folder with default app.
func OpenPath(path string) error {
	return exec.Command("cmd", "/c", "start", "", path).Start()
}
