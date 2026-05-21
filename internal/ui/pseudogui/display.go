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
	routeLabels := map[int]string{
		0: "manual", 1: "ru_direct", 2: "ru_blocked_ai", 3: "wg_over_wl_tunnel",
	}
	rq := s.RouteQuick
	if lbl, ok := routeLabels[rq]; ok {
		io.Line(fmt.Sprintf("Маршрут: %d (%s)", rq, lbl))
	} else {
		io.Line(fmt.Sprintf("Маршрут: %d", rq))
	}
	io.Line(fmt.Sprintf("Режим: %s  порт: %d", mode, s.MixedPort))
	if s.WLBuiltinConnectEnabled {
		io.Line("WL builtin fallback: вкл (H22 trojan rescue)")
	} else {
		io.Line("WL builtin fallback: выкл (только подписки)")
	}
	if s.MultipathEnabled {
		io.Line(fmt.Sprintf("Multipath: вкл  preset=%s  wl_emergency=%v", s.MultipathPreset, s.MultipathWLEmergencyOnly))
	} else {
		io.Line("Multipath: выкл")
	}
	if c.ProbeText != "" {
		io.Line(c.ProbeText)
	}
	if c.MultipathText != "" {
		io.Line(c.MultipathText)
	}
}
