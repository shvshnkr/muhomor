//go:build cgo

package fyneapp

import (
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
	if c.State == apiclient.StateConnecting || c.State == apiclient.StateStopping {
		return true
	}
	return false
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
