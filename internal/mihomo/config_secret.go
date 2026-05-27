package mihomo

import (
	"os"
	"strings"
)

// SecretFromConfig reads the secret: field from a mihomo YAML config.
func SecretFromConfig(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "secret:") {
			continue
		}
		v := strings.TrimSpace(strings.TrimPrefix(line, "secret:"))
		v = strings.Trim(v, `"'`)
		return v
	}
	return ""
}
