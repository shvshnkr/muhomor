package pseudogui

import (
	"context"
	"strconv"
	"strings"

	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/ui/console"
)

var settingsMenuLabels = []string{
	"[1] Показать текущие",
	"[2] Mixed port",
	"[3] Режим proxy / vpn",
	"[4] Режим: один туннель / пул load-balance",
	"[5] Пул PROXY_BULK вкл/выкл",
	"[6] Стратегия пула: Sticky / Consistent",
	"[7] Мин. туннелей в пуле",
	"[8] Макс. туннелей (0 = по preset)",
	"[9] Интервал recovery, сек",
	"[0] Multipath вкл/выкл",
	"[p] Multipath preset (low/normal/high)",
	"[w] WL только при деградации подписок",
	"[t] WL builtin trojan — аварийный fallback (H22)",
	"[b] Назад",
}

// RunSettingsMenu interactive settings editor (parity with fyneapp settings_tab).
func RunSettingsMenu(ctx context.Context, app *appcore.App, io *console.IO) int {
	for {
		idx, ok := RunListPicker(ctx, io, "Настройки (расширенные)", settingsMenuLabels, 0)
		if !ok {
			return 0
		}
		switch idx {
		case 0:
			if err := app.ShowSettings(ctx); err != nil {
				io.Line("→ " + err.Error())
				return 1
			}
		case 1:
			if code := editMixedPort(ctx, app, io); code != 0 {
				return code
			}
		case 2:
			if code := editServiceMode(ctx, app, io); code != 0 {
				return code
			}
		case 3:
			if code := toggleAggregationMode(ctx, app, io); code != 0 {
				return code
			}
		case 4:
			if code := toggleBulkEnabled(ctx, app, io); code != 0 {
				return code
			}
		case 5:
			if code := editBulkLBStrategy(ctx, app, io); code != 0 {
				return code
			}
		case 6:
			if code := editBulkMinLegs(ctx, app, io); code != 0 {
				return code
			}
		case 7:
			if code := editBulkMaxLegs(ctx, app, io); code != 0 {
				return code
			}
		case 8:
			if code := editBulkRecovery(ctx, app, io); code != 0 {
				return code
			}
		case 9:
			if code := toggleMultipath(ctx, app, io); code != 0 {
				return code
			}
		case 10:
			if code := editMultipathPreset(ctx, app, io); code != 0 {
				return code
			}
		case 11:
			if code := toggleMultipathWL(ctx, app, io); code != 0 {
				return code
			}
		case 12:
			if code := toggleWLBuiltinConnect(ctx, app, io); code != 0 {
				return code
			}
		case 13:
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
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	cur := 0
	if set.ServiceMode == appcore.ServiceModeVPN {
		cur = 1
	}
	opts := []string{"Proxy (mixed-port)", "VPN (TUN)"}
	idx, ok := RunListPicker(ctx, io, "Режим сервиса", opts, cur)
	if !ok {
		return 0
	}
	switch idx {
	case 0:
		set.ServiceMode = appcore.ServiceModeProxy
		set.TunEnable = false
	case 1:
		set.ServiceMode = appcore.ServiceModeVPN
		set.TunEnable = true
	}
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line("Сохранено: " + set.ServiceMode)
	return 0
}

func toggleAggregationMode(ctx context.Context, app *appcore.App, io *console.IO) int {
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	if set.AggregationMode == store.AggregationModeFlowAggregate {
		set.AggregationMode = store.AggregationModeLegacy
		set.BulkEnabled = false
	} else {
		set.AggregationMode = store.AggregationModeFlowAggregate
	}
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	if set.AggregationMode == store.AggregationModeFlowAggregate {
		io.Line("Режим: пул (flow_aggregate). Включите bulk [5] и переподключитесь.")
	} else {
		io.Line("Режим: один туннель (переподключитесь)")
	}
	return 0
}

func toggleBulkEnabled(ctx context.Context, app *appcore.App, io *console.IO) int {
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	if set.AggregationMode != store.AggregationModeFlowAggregate {
		io.Line("Сначала включите пул load-balance ([4]).")
		return 0
	}
	set.BulkEnabled = !set.BulkEnabled
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	if set.BulkEnabled {
		io.Line("Пул PROXY_BULK включён.")
	} else {
		io.Line("Пул выключен — один сервер (PROXY).")
	}
	return 0
}

func editBulkLBStrategy(ctx context.Context, app *appcore.App, io *console.IO) int {
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	if set.AggregationMode != store.AggregationModeFlowAggregate || !set.BulkEnabled {
		io.Line("Нужен включённый пул load-balance ([4]+[5]).")
		return 0
	}
	curIdx := 0
	if store.NormalizeBulkLBStrategy(set.BulkLBStrategy) == store.BulkLBConsistentHash {
		curIdx = 1
	}
	opts := []string{
		"Sticky (~10 мин на приложение)",
		"Consistent (по домену/IP)",
	}
	idx, ok := RunListPicker(ctx, io, "Стратегия пула", opts, curIdx)
	if !ok {
		return 0
	}
	switch idx {
	case 0:
		set.BulkLBStrategy = store.BulkLBStickySessions
	case 1:
		set.BulkLBStrategy = store.BulkLBConsistentHash
	}
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line("Стратегия сохранена. Переподключитесь [3].")
	return 0
}

func editBulkMinLegs(ctx context.Context, app *appcore.App, io *console.IO) int {
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line(fmtInt("Текущий bulk_min", set.BulkMinHealthyLegs))
	raw, err := io.ReadLine("Мин. туннелей 1..32 (Enter — отмена): ")
	if err != nil || strings.TrimSpace(raw) == "" {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 || n > 32 {
		io.Line("Нужно 1..32")
		return 1
	}
	set.BulkMinHealthyLegs = n
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line("Сохранено.")
	return 0
}

func editBulkMaxLegs(ctx context.Context, app *appcore.App, io *console.IO) int {
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line(fmtInt("Текущий bulk_max", set.BulkMaxLegs) + " (0 = preset low=3 normal=6 high=8)")
	raw, err := io.ReadLine("Новое значение 0..32 (Enter — отмена): ")
	if err != nil || strings.TrimSpace(raw) == "" {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 || n > 32 {
		io.Line("Нужно 0..32")
		return 1
	}
	set.BulkMaxLegs = n
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line("Сохранено. Переподключитесь при активном пуле.")
	return 0
}

func editBulkRecovery(ctx context.Context, app *appcore.App, io *console.IO) int {
	set, err := app.Config.LoadSettings(ctx)
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line(fmtInt("Текущий bulk_recovery", set.BulkRecoverySeconds))
	raw, err := io.ReadLine("Интервал 5..600 сек (Enter — отмена): ")
	if err != nil || strings.TrimSpace(raw) == "" {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 5 || n > 600 {
		io.Line("Нужно 5..600")
		return 1
	}
	set.BulkRecoverySeconds = n
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line("Сохранено.")
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
		io.Line("Сначала включите multipath ([0]).")
		return 0
	}
	opts := []string{"low", "normal", "high"}
	cur := 1
	for i, o := range opts {
		if stringsTrimLower(set.MultipathPreset) == o {
			cur = i
			break
		}
	}
	idx, ok := RunListPicker(ctx, io, "Multipath preset", opts, cur)
	if !ok {
		return 0
	}
	set.MultipathPreset = opts[idx]
	if _, err := app.Config.SaveSettings(ctx, set); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line("Preset сохранён: " + opts[idx])
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
