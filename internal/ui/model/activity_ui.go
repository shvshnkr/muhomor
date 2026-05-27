package model

import "strings"

// ActivityDuplicatesLabel reports whether activity repeats the status title (hide in pill).
func ActivityDuplicatesLabel(title, activity string) bool {
	title = strings.TrimSpace(title)
	activity = strings.TrimSpace(activity)
	if activity == "" {
		return true
	}
	if activity == title {
		return true
	}
	if strings.HasPrefix(activity, "Подключение") && strings.HasPrefix(title, "Подключение") {
		return true
	}
	return false
}

// FilterActivityForDisplay returns activity unless it duplicates the status label.
func FilterActivityForDisplay(title, activity string) string {
	if ActivityDuplicatesLabel(title, activity) {
		return ""
	}
	return strings.TrimSpace(activity)
}

// ConnectStatusTitle is the primary status line for simple-mode UI.
func ConnectStatusTitle(connected, connecting bool, errorText string) string {
	switch {
	case connected:
		return "Подключено"
	case connecting:
		return "Подключение…"
	case errorText != "":
		return "Ошибка"
	default:
		return "Отключено"
	}
}
