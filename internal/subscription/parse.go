package subscription

import (
	"bufio"
	"io"
	"strings"
)

// ParseLines extracts supported proxy URIs from subscription body.
func ParseLines(r io.Reader) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "://"); idx > 0 {
			line = line[idx:]
		}
		if strings.Contains(line, "://") && UnsupportedReason(line) == "" {
			out = append(out, line)
		}
	}
	return out, sc.Err()
}

// ParseVLESSLines is an alias for backward compatibility.
func ParseVLESSLines(r io.Reader) ([]string, error) {
	return ParseLines(r)
}
