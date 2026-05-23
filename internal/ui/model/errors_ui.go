package model

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// FriendlyConnectError turns daemon/API errors into short Russian text for UI.
func FriendlyConnectError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "Подключение отменено"
	}
	msg := extractAPIError(err.Error())
	switch {
	case strings.Contains(msg, "context canceled"), strings.Contains(msg, "context cancelled"),
		strings.Contains(msg, "connect aborted"):
		return "Подключение отменено"
	case strings.Contains(msg, "all subscription servers failed"):
		return "Нет доступных серверов в подписках.\n\n" +
			"• Расширенный режим → Настройки → включите «WL builtin trojan»\n" +
			"• или добавьте рабочую подписку в Конфигурация → Обновить подписку"
	case strings.Contains(msg, "no profiles"):
		return "Нет профилей. Добавьте подписку или импортируйте серверы (Конфигурация)."
	case strings.Contains(msg, "daemon not running"):
		return "Демон не запущен. Перезапустите GUI или Настройки → Запустить демон."
	default:
		if msg != "" {
			return msg
		}
		return err.Error()
	}
}

func extractAPIError(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, `{"error"`); i >= 0 {
		var body struct {
			Error string `json:"error"`
		}
		if json.Unmarshal([]byte(raw[i:]), &body) == nil && body.Error != "" {
			return body.Error
		}
	}
	if i := strings.Index(raw, ": "); i >= 0 && strings.HasPrefix(raw, "5") {
		rest := strings.TrimSpace(raw[i+2:])
		if j := strings.Index(rest, `{"error"`); j >= 0 {
			return extractAPIError(rest[j:])
		}
		return rest
	}
	return raw
}

// ErrConnectFailed wraps connect errors for dialogs (implements error).
type ErrConnectFailed struct{ Msg string }

func (e *ErrConnectFailed) Error() string { return e.Msg }

func NewConnectError(err error) error {
	if err == nil {
		return nil
	}
	return &ErrConnectFailed{Msg: FriendlyConnectError(err)}
}

func AsConnectError(err error) string {
	var ce *ErrConnectFailed
	if errors.As(err, &ce) {
		return ce.Msg
	}
	return FriendlyConnectError(err)
}
