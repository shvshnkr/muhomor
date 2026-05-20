package subscription

import "strings"

// SupportedSchemes for MVP + Phase 3 explicit list.
var SupportedSchemes = map[string]bool{
	"vless":     true,
	"trojan":    true,
	"hysteria":  true,
	"hysteria2": true,
}

// UnsupportedReason returns empty if line may be imported.
func UnsupportedReason(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "empty"
	}
	if idx := strings.Index(line, "://"); idx > 0 {
		sch := strings.ToLower(line[:idx])
		if SupportedSchemes[sch] {
			return ""
		}
		return "protocol_unsupported:" + sch
	}
	return ""
}
