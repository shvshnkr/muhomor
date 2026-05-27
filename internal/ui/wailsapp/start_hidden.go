package wailsapp

import (
	"os"
	"strings"
)

// StartHiddenEnabled returns true when GUI should start hidden (flag or env).
// MUHOMOR_GUI_START_HIDDEN: 1, true, yes (case-insensitive).
func StartHiddenEnabled(flag bool) bool {
	if flag {
		return true
	}
	v := strings.TrimSpace(os.Getenv("MUHOMOR_GUI_START_HIDDEN"))
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
