package pseudogui

import (
	"fmt"

	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/ui/console"
	"github.com/muhomor/muhomor/internal/ui/model"
)

func printStatus(io *console.IO, c model.ConnectionUI, s model.SettingsUI) {
	if c.ErrorText != "" {
		io.Line("Ошибка: " + c.ErrorText)
	}
	if c.Busy && c.ActivityText != "" {
		io.Line("… " + c.ActivityText)
	} else if c.ActivityText != "" {
		io.Line("Активность: " + c.ActivityText)
	}
	io.Line(fmt.Sprintf("Состояние: %s  connected=%v", c.State, c.Connected))
	if c.Connected {
		io.Line(fmt.Sprintf("Профиль: %s", c.ProfileName))
		io.Line(fmt.Sprintf("Прокси: %s", c.ProxyName))
	}
	mode := "proxy (mixed-port)"
	if s.ServiceMode == appcore.ServiceModeVPN {
		mode = "vpn (TUN)"
	}
	io.Line(fmt.Sprintf("Режим: %s  порт: %d", mode, s.MixedPort))
}
