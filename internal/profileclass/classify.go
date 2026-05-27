package profileclass

import (
	"strings"

	"github.com/muhomor/muhomor/internal/store"
)

// IsRuExitMarked reports RU-exit profiles (DB flag or name heuristic).
func IsRuExitMarked(p store.Profile) bool {
	if p.RuExitMarked {
		return true
	}
	return RuExitMarkedByName(p.Name)
}

// RuExitMarkedByName detects RU-exit from subscription display name.
func RuExitMarkedByName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	if strings.Contains(n, "russia") || strings.Contains(n, "🇷🇺") {
		return true
	}
	return strings.Contains(n, " ru ") || strings.HasPrefix(n, "ru ") ||
		strings.HasSuffix(n, " ru") || strings.Contains(n, "[ru]")
}

// IsBLSubscriptionMarked reports optional [bl] tag in name (ranking tag only).
func IsBLSubscriptionMarked(p store.Profile) bool {
	return strings.Contains(strings.ToLower(p.Name), "[bl]")
}

// IsBLModeUplink reports BL ISP uplink nodes (ru_exit and/or subscription [bl] tag).
func IsBLModeUplink(p store.Profile) bool {
	return IsRuExitMarked(p) || IsBLSubscriptionMarked(p)
}
