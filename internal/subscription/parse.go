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
		line = extractProxyURI(line)
		if strings.Contains(line, "://") && UnsupportedReason(line) == "" {
			out = append(out, line)
		}
	}
	return out, sc.Err()
}

// extractProxyURI keeps full scheme://… URIs; strips only junk *before* a known scheme.
func extractProxyURI(line string) string {
	lower := strings.ToLower(line)
	for sch := range SupportedSchemes {
		marker := sch + "://"
		if i := strings.Index(lower, marker); i >= 0 {
			return line[i:]
		}
	}
	return line
}

// ParseVLESSLines is an alias for backward compatibility.
func ParseVLESSLines(r io.Reader) ([]string, error) {
	return ParseLines(r)
}
