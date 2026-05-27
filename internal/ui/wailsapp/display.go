package wailsapp

import (
	"fmt"
	"strings"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/ui/model"
)

func sanitizeConnectionForQuit(dto ConnectionDTO) ConnectionDTO {
	dto.ErrorText = ""
	dto.ActivityText = ""
	dto.PingError = false
	dto.Busy = false
	dto.Connecting = false
	if dto.Connected {
		dto.StatusTitle = "Подключено"
	} else {
		dto.StatusTitle = "Отключено"
	}
	return dto
}

func connectionDTO(c model.ConnectionUI) ConnectionDTO {
	connecting := uiConnecting(c)
	dto := ConnectionDTO{
		State:        string(c.State),
		Connected:    c.Connected,
		ConnectedVerified: c.ConnectedVerified,
		ConnectedDegraded: c.ConnectedDegraded,
		VerificationPhase: c.VerificationPhase,
		ProfileID:    c.ProfileID,
		ProfileName:  c.ProfileName,
		ProxyName:    c.ProxyName,
		Busy:         c.Busy,
		Connecting:   connecting,
		ErrorText:    c.ErrorText,
		ConnectLabel: connectLabel(c),
		ConnectDanger: c.Connected && !c.Busy,
		PingEnabled:  c.Connected,
	}
	if len(c.BulkMembers) > 0 && c.Connected {
		dto.PingLabel = "Ping всех"
	} else if c.Connected {
		dto.PingLabel = "Ping"
	} else {
		dto.PingLabel = "Ping"
	}
	dto.StatusTitle, dto.ActivityText = statusTexts(c, connecting)
	dto.ProbeText = probeBlock(c)
	showPing := c.Connected && !c.Busy && !connecting
	switch {
	case showPing && c.LastPingMs > 0:
		dto.PingResult = fmt.Sprintf("%d ms", c.LastPingMs)
	case showPing && c.LastPingError != "":
		dto.PingResult = model.FriendlyDelayError(c.LastPingError)
		dto.PingError = true
	case c.Connected || connecting:
		dto.PingResult = "…"
	default:
		dto.PingResult = "—"
	}
	dto.TrafficUp = c.TrafficUp
	dto.TrafficDown = c.TrafficDown
	return dto
}

func connectLabel(c model.ConnectionUI) string {
	if c.Busy && !c.Connected {
		return "Отменить"
	}
	if c.Connected {
		return "Отключить"
	}
	return "Подключить"
}

func statusTexts(c model.ConnectionUI, connecting bool) (title, activity string) {
	switch {
	case c.VerificationPhase == "no_live_servers":
		return "Нет живых серверов", "После проверки живых серверов не найдено"
	case c.ConnectedDegraded:
		return "Подключено (degraded)", model.FilterActivityForDisplay("Подключено (degraded)", c.ActivityText)
	case c.Connected:
		title = "Подключено"
		activity = model.FilterActivityForDisplay(title, c.ActivityText)
	case connecting:
		title = "Подключение…"
		activity = model.FilterActivityForDisplay(title, c.ActivityText)
	case c.ErrorText != "":
		return "Ошибка", c.ErrorText
	default:
		title = "Отключено"
		if c.ActivityText != "" && !isStartupActivity(c.ActivityText) {
			activity = c.ActivityText
		}
	}
	return title, activity
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

func uiConnecting(c model.ConnectionUI) bool {
	if c.Connected {
		return false
	}
	if c.Busy {
		return true
	}
	return c.State == apiclient.StateConnecting || c.State == apiclient.StateStopping
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
		strings.HasPrefix(text, "Сервер нестабилен") ||
		strings.HasPrefix(text, "Скачивание geo-базы") ||
		text == "Geo-база готова"
}

func settingsDTO(s model.SettingsUI) SettingsDTO {
	return SettingsDTO{
		ServiceMode:              s.ServiceMode,
		MixedPort:                s.MixedPort,
		RouteQuick:               s.RouteQuick,
		MultipathEnabled:         s.MultipathEnabled,
		MultipathPreset:          s.MultipathPreset,
		MultipathWLEmergencyOnly: s.MultipathWLEmergencyOnly,
		WLBuiltinConnectEnabled:  s.WLBuiltinConnectEnabled,
		UIKeepErrorsOnScreen:     s.UIKeepErrorsOnScreen,
	}
}
