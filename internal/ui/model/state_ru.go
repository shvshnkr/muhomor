package model

import "github.com/muhomor/muhomor/internal/apiclient"

// ServiceStateRU returns a Russian label for daemon service state (API values stay English).
func ServiceStateRU(s apiclient.ServiceState) string {
	switch s {
	case apiclient.StateIdle, "":
		return "Ожидание"
	case apiclient.StateConnecting:
		return "Подключение"
	case apiclient.StateConnected:
		return "Подключено"
	case apiclient.StateStopping:
		return "Отключение"
	case apiclient.StateStopped:
		return "Остановлено"
	default:
		return string(s)
	}
}
