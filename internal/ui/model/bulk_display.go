package model

import (
	"fmt"
	"strings"
)

// BulkMembersSummary returns a one-line pool health summary.
func BulkMembersSummary(members []BulkMemberUI) string {
	if len(members) == 0 {
		return ""
	}
	ok := 0
	for _, m := range members {
		if m.DelayMs > 0 && m.Error == "" {
			ok++
		}
	}
	return fmt.Sprintf("живых %d/%d", ok, len(members))
}

// FormatBulkMembersTable renders members for terminal or multi-line UI.
func FormatBulkMembersTable(members []BulkMemberUI) string {
	if len(members) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Пул load-balance:\n")
	for i, m := range members {
		label := m.Name
		if label == "" {
			label = m.Tag
		}
		status := "—"
		switch {
		case m.DelayMs > 0 && m.Error == "":
			status = fmt.Sprintf("%d ms", m.DelayMs)
		case m.Error != "":
			status = "ошибка: " + m.Error
		}
		fmt.Fprintf(&b, "  %2d. %-28s %s\n", i+1, truncateRunes(label, 28), status)
	}
	return strings.TrimRight(b.String(), "\n")
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
