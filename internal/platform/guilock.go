package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// AcquireGUILock ensures a single muhomor-gui instance per data directory.
func AcquireGUILock(dataDir string) (release func(), err error) {
	path := filepath.Join(dataDir, "gui.lock")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			hint := "закройте через трей → Выход или завершите muhomor-gui.exe в диспетчере задач"
			if pid := readLockPID(path); pid > 0 {
				return nil, fmt.Errorf("muhomor-gui уже запущен (PID %d). %s", pid, hint)
			}
			return nil, fmt.Errorf("muhomor-gui уже запущен (lock: %s). %s", path, hint)
		}
		return nil, err
	}
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
	return func() {
		_ = f.Close()
		_ = os.Remove(path)
	}, nil
}

// StaleGUILock removes lock file if PID inside is not running (after crash).
func StaleGUILock(dataDir string) {
	path := filepath.Join(dataDir, "gui.lock")
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		_ = os.Remove(path)
		return
	}
	if processAlive(pid) {
		return
	}
	_ = os.Remove(path)
}

func readLockPID(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return 0
	}
	return pid
}
