package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// Layout mirrors Dahusim desktop data dirs (simplified).
type Layout struct {
	DataDir    string
	CacheDir   string
	RuntimeDir string
}

func Default(base string) Layout {
	if base == "" {
		base = defaultDataDir()
	}
	cache := filepath.Join(base, "cache")
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		runtime = filepath.Join(base, "run")
	} else {
		runtime = filepath.Join(runtime, "muhomor")
	}
	return Layout{
		DataDir:    base,
		CacheDir:   cache,
		RuntimeDir: runtime,
	}
}

func (l Layout) DBPath() string {
	return filepath.Join(l.DataDir, "muhomor.db")
}

func (l Layout) ConfigPath() string {
	return filepath.Join(l.RuntimeDir, "config.yaml")
}

func (l Layout) MihomoDir() string {
	return filepath.Join(l.RuntimeDir, "mihomo")
}

func (l Layout) ControlStatusFile() string {
	return filepath.Join(l.CacheDir, "desktop-control-status.txt")
}

func (l Layout) SocketPath() string {
	return filepath.Join(l.RuntimeDir, "muhomor.sock")
}

func (l Layout) Ensure() error {
	for _, d := range []string{l.DataDir, l.CacheDir, l.RuntimeDir, l.MihomoDir()} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func defaultDataDir() string {
	if runtime.GOOS == "windows" {
		if app := os.Getenv("LOCALAPPDATA"); app != "" {
			return filepath.Join(app, "muhomor")
		}
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "muhomor")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "muhomor")
}
