package mihomo

import (
	"os"
	"path/filepath"
)

// ResolveBin returns mihomo executable: env, bin/ next to muhomor, then PATH name.
func ResolveBin() string {
	if v := os.Getenv("MUHOMOR_MIHOMO_BIN"); v != "" {
		return v
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for _, name := range []string{
			filepath.Join(dir, "bin", "mihomo.exe"),
			filepath.Join(dir, "bin", "mihomo"),
			filepath.Join(dir, "mihomo.exe"),
			filepath.Join(dir, "mihomo"),
		} {
			if st, err := os.Stat(name); err == nil && !st.IsDir() {
				return name
			}
		}
	}
	return "mihomo"
}
