package pseudogui

import (
	"context"
	"strconv"
	"strings"

	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/ui/console"
)

// RunSettingsMenu interactive settings editor (parity with fyneapp settings_tab).
func RunSettingsMenu(ctx context.Context, app *appcore.App, io *console.IO) int {
	for {
		io.Line("")
		io.Line("--- Настройки (расширенные) ---")
		io.Line("[1] Показать текущие")
		io.Line("[2] Mixed port")
		io.Line("[3] Режим proxy / vpn")
		io.Line("[4] Multipath вкл/выкл")
		io.Line("[5] Multipath preset (low/normal/high)")
		io.Line("[6] WL только при деградации подписок (multipath)")
		io.Line("[7] WL builtin trojan — аварийный fallback (H22)")
		io.Line("[b] Назад в главное меню")
		choice, err := io.ReadLine("Выбор: ")
		if err != nil {
			return 0
		}
		switch stringsTrimLower(choice) {
		case "1":
			if err := app.ShowSettings(ctx); err != nil {
				io.Line("→ " + err.Error())
				return 1
			}
		case "2":
			if code := editMixedPort(ctx, app, io); code != 0 {
				return code
			}
		case "3":
			if code := editServiceMode(ctx, app, io); code != 0 {
				return code
			}
		case "4":
			if code := toggleMultipath(ctx, app, io); code != 0 {
				return code
			}
		case "5":
			if code := editMultipathPreset(ctx, app, io); code != 0 {
				return code
			}
		case "6":
			if code := toggleMultipathWL(ctx, app, io); code != 0 {
				return code
			}
		case "7":
			if code := toggleWLBuiltinConnect(ctx, app, io); code != 0 {
				return code
			}
		case "b", "back", "назад":
			return 0
		default:
			io.Line("Неизвестная команда")
		}
	}
}

func editMixedPort(ctx context.Context, app *appcore.App, io *console.IO) int {
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line(fmtInt("Текущий mixed_port", set.MixedPort))
	raw, err := io.ReadLine("Новый порт (Enter — отмена): ")
	if err != nil || strings.TrimSpace(raw) == "" {
		return 0
	}
	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || port <= 0 || port > 65535 {
		io.Line("Нужно число 1..65535")
		return 1
	}
	set.MixedPort = port
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line("Сохранено. Reload [5] или переподключение [3].")
	return 0
}

func editServiceMode(ctx context.Context, app *appcore.App, io *console.IO) int {
	io.Line("1 = proxy (mixed-port), 2 = vpn (TUN)")
	raw, err := io.ReadLine("Режим (Enter — отмена): ")
	if err != nil || strings.TrimSpace(raw) == "" {
		return 0
	}
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	switch strings.TrimSpace(raw) {
	case "1":
		set.ServiceMode = appcore.ServiceModeProxy
		set.TunEnable = false
	case "2":
		set.ServiceMode = appcore.ServiceModeVPN
		set.TunEnable = true
	default:
		io.Line("Нужно 1 или 2")
		return 1
	}
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line("Сохранено: " + set.ServiceMode)
	return 0
}

func toggleMultipath(ctx context.Context, app *appcore.App, io *console.IO) int {
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	set.MultipathEnabled = !set.MultipathEnabled
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	if set.MultipathEnabled {
		io.Line("Multipath включён (умный выбор каналов по goodput).")
	} else {
		io.Line("Multipath выключен (классический selector по delay).")
	}
	return 0
}

func editMultipathPreset(ctx context.Context, app *appcore.App, io *console.IO) int {
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	if !set.MultipathEnabled {
		io.Line("Сначала включите multipath ([4]).")
		return 0
	}
	io.Line(fmtStr("Текущий preset", set.MultipathPreset))
	raw, err := io.ReadLine("low / normal / high (Enter — отмена): ")
	if err != nil || strings.TrimSpace(raw) == "" {
		return 0
	}
	p := stringsTrimLower(strings.TrimSpace(raw))
	if p != "low" && p != "normal" && p != "high" {
		io.Line("Нужно low, normal или high")
		return 1
	}
	set.MultipathPreset = p
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line("Preset сохранён: " + p)
	return 0
}

func toggleWLBuiltinConnect(ctx context.Context, app *appcore.App, io *console.IO) int {
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	set.WLBuiltinConnectEnabled = !set.WLBuiltinConnectEnabled
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	if set.WLBuiltinConnectEnabled {
		io.Line("WL builtin включён: при мёртвых подписках возможен H22-retry на trojan.")
	} else {
		io.Line("WL builtin выключен: connect только по подпискам (рекомендуется).")
	}
	return 0
}

func toggleMultipathWL(ctx context.Context, app *appcore.App, io *console.IO) int {
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	set.MultipathWLEmergencyOnly = !set.MultipathWLEmergencyOnly
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	if set.MultipathWLEmergencyOnly {
		io.Line("WL builtin только при деградации подписок (рекомендуется).")
	} else {
		io.Line("WL может входить в multipath-пул вместе с подписками.")
	}
	return 0
}

func fmtInt(label string, v int) string {
	return label + ": " + strconv.Itoa(v)
}

func fmtStr(label, v string) string {
	if v == "" {
		v = "(пусто)"
	}
	return label + ": " + v
}
