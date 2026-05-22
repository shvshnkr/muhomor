package paths

import (
	"os"
	"path/filepath"
	"strings"
)

// PortableMixedPort is the default mixed inbound port for the Windows portable kit.
const PortableMixedPort = 2181

// ResolveDataDir picks the data directory: -dir / MUHOMOR_DATA_DIR / kit-local data / OS default.
func ResolveDataDir(explicit string) string {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		return explicit
	}
	if v := strings.TrimSpace(os.Getenv("MUHOMOR_DATA_DIR")); v != "" {
		return v
	}
	if root, ok := PortableKitRoot(); ok {
		return filepath.Join(root, "data")
	}
	return defaultDataDir()
}

// PortableKitRoot is the directory of a self-contained muhomor-kit (exe + bin/mihomo).
func PortableKitRoot() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	root, err := filepath.Abs(filepath.Dir(exe))
	if err != nil {
		return "", false
	}
	if portableKitMarkersPresent(root) {
		return root, true
	}
	return "", false
}

func portableKitMarkersPresent(root string) bool {
	if fileExists(filepath.Join(root, ".muhomor-portable")) {
		return true
	}
	if b, err := os.ReadFile(filepath.Join(root, "VERSION.txt")); err == nil {
		if strings.Contains(string(b), "muhomor-kit") {
			return true
		}
	}
	// Windows kit layout from pack-windows-kit.ps1
	hasBinMihomo := fileExists(filepath.Join(root, "bin", "mihomo.exe")) ||
		fileExists(filepath.Join(root, "bin", "mihomo"))
	if !hasBinMihomo {
		return false
	}
	return fileExists(filepath.Join(root, "muhomor.exe")) ||
		fileExists(filepath.Join(root, "muhomor-gui.exe"))
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

// ApplyPortableKitEnv sets MUHOMOR_MIHOMO_BIN when running from a portable kit folder.
func ApplyPortableKitEnv() {
	root, ok := PortableKitRoot()
	if !ok {
		return
	}
	if strings.TrimSpace(os.Getenv("MUHOMOR_MIHOMO_BIN")) != "" {
		return
	}
	for _, name := range []string{"mihomo.exe", "mihomo"} {
		p := filepath.Join(root, "bin", name)
		if fileExists(p) {
			_ = os.Setenv("MUHOMOR_MIHOMO_BIN", p)
			return
		}
	}
}

// DefaultMixedPort returns requested port, or kit 2181, or desktop default 7890.
func DefaultMixedPort(requested int) int {
	if requested > 0 {
		return requested
	}
	if _, ok := PortableKitRoot(); ok {
		return PortableMixedPort
	}
	return 7890
}
