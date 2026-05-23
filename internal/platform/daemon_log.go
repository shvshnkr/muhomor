package platform

import (
	"os"
	"os/exec"
	"path/filepath"
)

// AttachDaemonLog redirects child stderr to cache/daemon-debug.err.log (kit-friendly).
func AttachDaemonLog(cmd *exec.Cmd, cacheDir string) {
	if cmd == nil || cacheDir == "" {
		return
	}
	_ = os.MkdirAll(cacheDir, 0o700)
	path := filepath.Join(cacheDir, "daemon-debug.err.log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		cmd.Stderr = f
	}
}
