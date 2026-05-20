package subscription

import "strings"

// Scheme returns URI scheme or empty.
func Scheme(line string) string {
	if i := strings.Index(line, "://"); i > 0 {
		return strings.ToLower(line[:i])
	}
	return ""
}
