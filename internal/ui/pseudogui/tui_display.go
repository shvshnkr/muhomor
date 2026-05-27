package pseudogui

import (
	"fmt"
	"strings"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/ui/model"
)

func uiConnecting(c model.ConnectionUI) bool {
	if c.Connected {
		return false
	}
	if c.Busy {
		return true
	}
	return c.State == apiclient.StateConnecting || c.State == apiclient.StateStopping
}

func statusTitle(c model.ConnectionUI) string {
	connecting := uiConnecting(c)
	switch {
	case c.Connected:
		return "Подключено"
	case c.ErrorText != "":
		return "Ошибка"
	case connecting:
		return "Подключение…"
	default:
		return "Отключено"
	}
}

func statusActivity(c model.ConnectionUI) string {
	connecting := uiConnecting(c)
	switch {
	case c.Connected, connecting:
		return c.ActivityText
	case c.ErrorText != "":
		return c.ErrorText
	default:
		if c.ActivityText != "" && !isStartupActivity(c.ActivityText) {
			return c.ActivityText
		}
	}
	return ""
}

func connectLabel(c model.ConnectionUI) string {
	if c.Busy && !c.Connected {
		return "ОТМЕНИТЬ"
	}
	if c.Connected {
		return "ОТКЛЮЧИТЬ"
	}
	return "ПОДКЛЮЧИТЬ"
}

func pingBadge(c model.ConnectionUI, pingBusy bool) string {
	if pingBusy {
		return "…"
	}
	if c.LastPingMs > 0 && (c.Connected || uiConnecting(c)) {
		return fmt.Sprintf("%d ms", c.LastPingMs)
	}
	if c.LastPingError != "" && (c.Connected || uiConnecting(c)) {
		return c.LastPingError
	}
	if c.Connected || uiConnecting(c) {
		return "…"
	}
	return "—"
}

func serverCardLines(c model.ConnectionUI) (title, caption string, flag string) {
	if c.Connected {
		d := model.ParseServerDisplay(c.ProfileName)
		return d.ServerName, countryCaption(d), d.Flag
	}
	d := model.DisconnectedServerTitle(c.ProfileName)
	return d.ServerName, countryCaption(d), d.Flag
}

func countryCaption(d model.ServerDisplay) string {
	if d.ShowCountry {
		return d.Country
	}
	return ""
}

func probeBlock(c model.ConnectionUI) string {
	if c.ErrorText != "" {
		return ""
	}
	var parts []string
	if c.Connected {
		if label := model.ProxyDisplayLabel(c.ProfileName, c.ProxyName); label != "" {
			parts = append(parts, fmt.Sprintf("Прокси: %s", label))
		}
	}
	for _, line := range []string{c.ProbeText, c.StandbyText, c.MultipathText} {
		if line != "" {
			parts = append(parts, line)
		}
	}
	if table := model.FormatBulkMembersTable(c.BulkMembers); table != "" {
		parts = append(parts, table)
	}
	return strings.Join(parts, "\n")
}

func isStartupActivity(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	return strings.HasPrefix(text, "Запуск mihomo") ||
		strings.HasPrefix(text, "Подключение") ||
		strings.HasPrefix(text, "Переподключение") ||
		strings.HasPrefix(text, "Обновление конфигурации") ||
		strings.HasPrefix(text, "Проверка соединения") ||
		strings.HasPrefix(text, "Сервер нестабилен")
}

func statusPillColor(c model.ConnectionUI) string {
	switch {
	case c.ErrorText != "":
		return ansiRed
	case uiConnecting(c):
		return ansiYellow
	case c.Connected:
		return ansiGreen
	default:
		return ansiMuted
	}
}

func statusMark(c model.ConnectionUI) string {
	switch {
	case c.ErrorText != "":
		return "✗"
	case uiConnecting(c):
		return "◐"
	case c.Connected:
		return "●"
	default:
		return "○"
	}
}
