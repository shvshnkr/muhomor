package pseudogui

import (
	"fmt"
	"strings"

	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/ui/console"
	"github.com/muhomor/muhomor/internal/ui/model"
)

// printStatusBanner compact always-visible status (before menu).
func printStatusBanner(io *console.IO, c model.ConnectionUI, s model.SettingsUI) {
	io.Line("")
	io.Line("┌─ Статус " + strings.Repeat("─", 48) + "┐")
	io.Line("│ " + statusHeadline(c))
	io.Line("│ " + statusDetails(c, s))
	if line := pingLine(c); line != "" {
		io.Line("│ " + line)
	}
	if c.ErrorText != "" {
		io.Line("│ ⚠ " + c.ErrorText)
	}
	io.Line("└" + strings.Repeat("─", 58) + "┘")
}

func statusHeadline(c model.ConnectionUI) string {
	mark := stateMark(c)
	state := model.ServiceStateRU(c.State)
	if c.Busy {
		state = "Подключение…"
	}
	line := fmt.Sprintf("%s %-14s", mark, state)
	if c.Connected {
		prof := c.ProfileName
		if prof == "" {
			prof = "—"
		}
		line += fmt.Sprintf(" │ %s", prof)
		if c.ProxyName != "" {
			line += " → " + c.ProxyName
		}
	}
	return padLine(line, 56)
}

func stateMark(c model.ConnectionUI) string {
	switch {
	case c.ErrorText != "":
		return "✗"
	case c.Busy:
		return "◐"
	case c.Connected:
		return "●"
	default:
		return "○"
	}
}

func statusDetails(c model.ConnectionUI, s model.SettingsUI) string {
	var parts []string
	mode := "proxy"
	if s.ServiceMode == appcore.ServiceModeVPN {
		mode = "vpn"
	}
	parts = append(parts, fmt.Sprintf("%s :%d", mode, s.MixedPort))
	if c.MultipathText != "" {
		parts = append(parts, c.MultipathText)
	} else if c.StandbyText != "" {
		parts = append(parts, c.StandbyText)
	} else if c.ProbeText != "" {
		parts = append(parts, c.ProbeText)
	}
	if len(c.BulkMembers) > 0 {
		parts = append(parts, "пул "+model.BulkMembersSummary(c.BulkMembers))
	}
	if c.Busy && c.ActivityText != "" {
		parts = append(parts, c.ActivityText)
	} else if !c.Busy && c.ActivityText != "" {
		parts = append(parts, c.ActivityText)
	}
	return padLine(strings.Join(parts, " │ "), 56)
}

func pingLine(c model.ConnectionUI) string {
	if c.LastPingMs > 0 {
		return padLine(fmt.Sprintf("Пинг: %d ms", c.LastPingMs), 56)
	}
	if c.LastPingError != "" {
		return padLine("Пинг: "+c.LastPingError, 56)
	}
	return ""
}

func padLine(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return string(r[:width])
	}
	return s + strings.Repeat(" ", width-len(r))
}

// printStatus full status dump (menu action [1]).
func printStatus(io *console.IO, c model.ConnectionUI, s model.SettingsUI) {
	printStatusBanner(io, c, s)
	io.Line("")
	if c.Connected {
		io.Line(fmt.Sprintf("Профиль: %s", c.ProfileName))
		io.Line(fmt.Sprintf("Прокси:  %s", model.ProxyDisplayLabel(c.ProfileName, c.ProxyName)))
	}
	routeLabels := map[int]string{
		0: "manual", 1: "ru_direct", 2: "ru_blocked_ai", 3: "wg_over_wl_tunnel",
	}
	rq := s.RouteQuick
	if lbl, ok := routeLabels[rq]; ok {
		io.Line(fmt.Sprintf("Маршрут: %d (%s)", rq, lbl))
	} else {
		io.Line(fmt.Sprintf("Маршрут: %d", rq))
	}
	if s.MultipathEnabled {
		io.Line(fmt.Sprintf("Multipath: вкл  preset=%s  wl_emergency=%v", s.MultipathPreset, s.MultipathWLEmergencyOnly))
	} else {
		io.Line("Multipath: выкл")
	}
	if s.WLBuiltinConnectEnabled {
		io.Line("WL builtin fallback: вкл")
	} else {
		io.Line("WL builtin fallback: выкл")
	}
	if table := model.FormatBulkMembersTable(c.BulkMembers); table != "" {
		io.Line("")
		io.Line(table)
	}
}

// printBulkMembers shows pool table (menu [b]).
func printBulkMembers(io *console.IO, c model.ConnectionUI) {
	if len(c.BulkMembers) == 0 {
		io.Line("Пул load-balance не активен. Подключитесь с режимом «пул туннелей» в настройках [s].")
		return
	}
	io.Line(model.FormatBulkMembersTable(c.BulkMembers))
	io.Line("Сводка: " + model.BulkMembersSummary(c.BulkMembers))
}
