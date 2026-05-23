//go:build cgo

package fyneapp

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/platform"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/ui/presenter"
)

const (
	lbUIModeSingle    = "Один туннель"
	lbUIModePool      = "Пул туннелей (load-balance)"
	lbUIStratSticky   = "Sticky"
	lbUIStratConsistent = "Consistent"
)

type settingsTab struct {
	content     fyne.CanvasObject
	w           fyne.Window
	app         *appcore.App
	ctx         context.Context
	opt         Options
	onBack      func()
	pres        *presenter.Presenter
	statusLabel *widget.Label
	hintLabel   *widget.Label
	mixedEntry  *widget.Entry
	modeRadio   *widget.RadioGroup
	poolModeRadio *widget.RadioGroup
	bulkOn        *widget.Check
	lbStrategyRadio *widget.RadioGroup
	strategyHint  *widget.Label
	bulkMinEntry  *widget.Entry
	bulkMaxEntry  *widget.Entry
	bulkRecEntry  *widget.Entry
	poolSection   fyne.CanvasObject
	multipathOn   *widget.Check
	mpPreset      *widget.Select
	mpWLEmerg     *widget.Check
	wlBuiltinOn   *widget.Check

	loading   bool
	saveMu    sync.Mutex
	saveTimer *time.Timer
	lastSaved apiclient.Settings
}

func newSettingsTab(w fyne.Window, app *appcore.App, ctx context.Context, opt Options, onBack func()) *settingsTab {
	s := &settingsTab{w: w, app: app, ctx: ctx, opt: opt, onBack: onBack}
	s.statusLabel = widget.NewLabel("")
	s.statusLabel.Wrapping = fyne.TextWrapWord
	s.hintLabel = hintLabel("Настройки сохраняются сами. После смены пула или стратегии переподключитесь. «Reload mihomo» — если меняли порт или режим Proxy/VPN на активном соединении.")
	s.mixedEntry = widget.NewEntry()
	s.mixedEntry.SetPlaceHolder("2181")
	applyTooltip(s.mixedEntry, "Локальный порт mixed-proxy (HTTP/SOCKS). При смене на активном соединении — Reload.")

	s.modeRadio = widget.NewRadioGroup([]string{"Proxy (mixed-port)", "VPN (TUN)"}, nil)

	s.poolModeRadio = widget.NewRadioGroup([]string{lbUIModeSingle, lbUIModePool}, nil)
	s.poolModeRadio.SetSelected(lbUIModePool)

	s.bulkOn = widget.NewCheck("Использовать пул PROXY_BULK", nil)
	s.bulkOn.SetChecked(true)
	applyTooltip(s.bulkOn, "Включить группу mihomo load-balance с несколькими туннелями. Выключено — один выбранный сервер (PROXY).")

	s.lbStrategyRadio = widget.NewRadioGroup([]string{lbUIStratSticky, lbUIStratConsistent}, nil)
	s.lbStrategyRadio.SetSelected(lbUIStratSticky)
	applyTooltip(s.lbStrategyRadio, "Как mihomo распределяет соединения между туннелями в пуле.")

	s.strategyHint = hintLabel(lbStrategyHintText(lbUIStratSticky))

	s.bulkMinEntry = widget.NewEntry()
	s.bulkMinEntry.SetPlaceHolder("2")
	s.bulkMaxEntry = widget.NewEntry()
	s.bulkMaxEntry.SetPlaceHolder("0 = по профилю")
	s.bulkRecEntry = widget.NewEntry()
	s.bulkRecEntry.SetPlaceHolder("60")

	s.multipathOn = widget.NewCheck("Умный выбор каналов (goodput, не только пинг)", nil)
	s.mpPreset = widget.NewSelect([]string{"low", "normal", "high"}, nil)
	s.mpPreset.SetSelected("normal")
	s.mpWLEmerg = widget.NewCheck("WL только при деградации подписок", nil)
	s.mpWLEmerg.SetChecked(true)
	s.wlBuiltinOn = widget.NewCheck("WL builtin trojan — аварийный fallback", nil)
	s.wlBuiltinOn.SetChecked(false)

	s.poolSection = container.NewVBox(
		s.bulkOn,
		widget.NewLabel("Стратегия балансировки"),
		s.lbStrategyRadio,
		s.strategyHint,
		formField("Минимум туннелей в пуле", s.bulkMinEntry,
			"Сколько серверов должно пройти проверку, чтобы пул включился. Обычно 2."),
		formField("Максимум туннелей (0 = авто)", s.bulkMaxEntry,
			"Верхняя граница пула: 3 / 6 / 8 в зависимости от профиля multipath (low/normal/high)."),
		formField("Проверка живости, сек", s.bulkRecEntry,
			"Как часто mihomo перепроверяет туннели в пуле. Меньше — быстрее убирает «мёртвые» серверы."),
	)

	backBtn := backButton("← Простой режим", func() { s.onBack() })
	reloadBtn := widget.NewButton("Reload mihomo", s.reloadNow)
	reloadBtn.Importance = widget.MediumImportance
	daemonStartBtn := widget.NewButton("Запустить демон", s.startDaemon)
	daemonStopBtn := widget.NewButton("Остановить демон", s.stopDaemon)
	daemonStopBtn.Importance = widget.DangerImportance

	body := container.NewVBox(
		sectionHeader("Настройки", "Порт, режим сервиса и выбор серверов"),
		s.hintLabel,
		formField("Mixed port", s.mixedEntry, ""),
		widget.NewLabel("Режим сервиса"),
		s.modeRadio,
		reloadBtn,
		widget.NewSeparator(),
		sectionHeader("Распределение нагрузки",
			"Пул туннелей mihomo (load-balance). Не путать с multipath: здесь трафик делится между серверами в одной группе PROXY_BULK."),
		widget.NewLabel("Режим маршрутизации"),
		s.poolModeRadio,
		s.poolSection,
		widget.NewSeparator(),
		sectionHeader("Multipath", "Какой сервер выбрать главным и кого включить в пул каналов."),
		s.multipathOn,
		widget.NewLabel("Профиль нагрузки"),
		s.mpPreset,
		s.mpWLEmerg,
		s.wlBuiltinOn,
		widget.NewSeparator(),
		sectionHeader("Демон", "Остановка завершает процесс muhomor --daemon, не только VPN."),
		container.NewHBox(daemonStartBtn, daemonStopBtn),
		vSpacer(4),
		surfaceCard(s.statusLabel, 0),
	)

	s.content = container.NewBorder(
		backBtn,
		nil, nil, nil,
		scrollContent(body),
	)
	s.wireAutoSave()
	return s
}

func lbStrategyHintText(selected string) string {
	switch selected {
	case lbUIStratConsistent:
		return "Consistent: все соединения к одному сайту (домену/IP) идут через один и тот же туннель. Удобно, если сайт чувствителен к смене IP."
	default:
		return "Sticky: соединения с одного приложения к одному адресату ~10 минут идут через один туннель. Обычно лучше для браузера, мессенджеров и speedtest."
	}
}

func (s *settingsTab) wireAutoSave() {
	schedule := func() { s.scheduleAutoSave() }
	s.poolModeRadio.OnChanged = func(string) {
		fyne.Do(s.updatePoolWidgets)
		schedule()
	}
	s.lbStrategyRadio.OnChanged = func(sel string) {
		fyne.Do(func() {
			s.strategyHint.SetText(lbStrategyHintText(sel))
		})
		schedule()
	}
	s.bulkOn.OnChanged = func(bool) { schedule() }
	s.bulkMinEntry.OnChanged = func(string) { schedule() }
	s.bulkMaxEntry.OnChanged = func(string) { schedule() }
	s.bulkRecEntry.OnChanged = func(string) { schedule() }
	s.multipathOn.OnChanged = func(bool) {
		fyne.Do(s.updateMultipathWidgets)
		schedule()
	}
	s.mpPreset.OnChanged = func(string) { schedule() }
	s.mpWLEmerg.OnChanged = func(bool) { schedule() }
	s.wlBuiltinOn.OnChanged = func(bool) { schedule() }
	s.modeRadio.OnChanged = func(string) { schedule() }
	s.mixedEntry.OnChanged = func(string) { schedule() }
}

func (s *settingsTab) bindPresenter(pres *presenter.Presenter) {
	s.pres = pres
}

func (s *settingsTab) load(ctx context.Context) {
	set, err := s.app.Config.LoadSettings(ctx)
	if err != nil {
		return
	}
	s.loading = true
	defer func() { s.loading = false }()
	fyne.Do(func() {
		s.lastSaved = set
		s.mixedEntry.SetText(strconv.Itoa(set.MixedPort))
		if set.ServiceMode == appcore.ServiceModeVPN {
			s.modeRadio.SetSelected("VPN (TUN)")
		} else {
			s.modeRadio.SetSelected("Proxy (mixed-port)")
		}
		if set.AggregationMode == store.AggregationModeFlowAggregate {
			s.poolModeRadio.SetSelected(lbUIModePool)
		} else {
			s.poolModeRadio.SetSelected(lbUIModeSingle)
		}
		s.bulkOn.SetChecked(set.BulkEnabled)
		switch store.NormalizeBulkLBStrategy(set.BulkLBStrategy) {
		case store.BulkLBConsistentHash:
			s.lbStrategyRadio.SetSelected(lbUIStratConsistent)
		default:
			s.lbStrategyRadio.SetSelected(lbUIStratSticky)
		}
		s.strategyHint.SetText(lbStrategyHintText(s.lbStrategyRadio.Selected))
		if set.BulkMinHealthyLegs > 0 {
			s.bulkMinEntry.SetText(strconv.Itoa(set.BulkMinHealthyLegs))
		} else {
			s.bulkMinEntry.SetText("2")
		}
		if set.BulkMaxLegs > 0 {
			s.bulkMaxEntry.SetText(strconv.Itoa(set.BulkMaxLegs))
		} else {
			s.bulkMaxEntry.SetText("")
		}
		if set.BulkRecoverySeconds > 0 {
			s.bulkRecEntry.SetText(strconv.Itoa(set.BulkRecoverySeconds))
		} else {
			s.bulkRecEntry.SetText("60")
		}
		s.multipathOn.SetChecked(set.MultipathEnabled)
		preset := set.MultipathPreset
		if preset == "" {
			preset = "normal"
		}
		s.mpPreset.SetSelected(preset)
		s.mpWLEmerg.SetChecked(set.MultipathWLEmergencyOnly)
		s.wlBuiltinOn.SetChecked(set.WLBuiltinConnectEnabled)
		s.updatePoolWidgets()
		s.updateMultipathWidgets()
	})
}

func (s *settingsTab) updatePoolWidgets() {
	pool := s.poolModeRadio.Selected == lbUIModePool
	if pool {
		s.poolSection.Show()
		s.bulkOn.Enable()
		s.lbStrategyRadio.Enable()
		s.bulkMinEntry.Enable()
		s.bulkMaxEntry.Enable()
		s.bulkRecEntry.Enable()
		if !s.multipathOn.Checked {
			s.multipathOn.SetChecked(true)
		}
	} else {
		s.poolSection.Hide()
		s.bulkOn.Disable()
		s.lbStrategyRadio.Disable()
		s.bulkMinEntry.Disable()
		s.bulkMaxEntry.Disable()
		s.bulkRecEntry.Disable()
	}
}

func (s *settingsTab) updateMultipathWidgets() {
	on := s.multipathOn.Checked
	if s.mpPreset != nil {
		if on {
			s.mpPreset.Enable()
		} else {
			s.mpPreset.Disable()
		}
	}
	if s.mpWLEmerg != nil {
		if on {
			s.mpWLEmerg.Enable()
		} else {
			s.mpWLEmerg.Disable()
		}
	}
}

func (s *settingsTab) scheduleAutoSave() {
	if s.loading {
		return
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	if s.saveTimer != nil {
		s.saveTimer.Stop()
	}
	s.saveTimer = time.AfterFunc(400*time.Millisecond, func() {
		fyne.Do(func() { s.persistSettings(false) })
	})
}

func (s *settingsTab) readForm() (apiclient.Settings, error) {
	port, err := strconv.Atoi(s.mixedEntry.Text)
	if err != nil || port <= 0 {
		port = s.lastSaved.MixedPort
		if port <= 0 {
			port = 2181
		}
	}
	set := s.lastSaved
	set.MixedPort = port
	if s.modeRadio.Selected == "VPN (TUN)" {
		set.ServiceMode = appcore.ServiceModeVPN
		set.TunEnable = true
	} else {
		set.ServiceMode = appcore.ServiceModeProxy
		set.TunEnable = false
	}
	if s.poolModeRadio.Selected == lbUIModePool {
		set.AggregationMode = store.AggregationModeFlowAggregate
		set.BulkEnabled = s.bulkOn.Checked
		set.MultipathEnabled = true
	} else {
		set.AggregationMode = store.AggregationModeLegacy
		set.BulkEnabled = false
	}
	if s.lbStrategyRadio.Selected == lbUIStratConsistent {
		set.BulkLBStrategy = store.BulkLBConsistentHash
	} else {
		set.BulkLBStrategy = store.BulkLBStickySessions
	}
	if n, err := strconv.Atoi(strings.TrimSpace(s.bulkMinEntry.Text)); err == nil && n > 0 {
		set.BulkMinHealthyLegs = n
	} else if set.BulkMinHealthyLegs <= 0 {
		set.BulkMinHealthyLegs = 2
	}
	if n, err := strconv.Atoi(strings.TrimSpace(s.bulkMaxEntry.Text)); err == nil && n > 0 {
		set.BulkMaxLegs = n
	} else {
		set.BulkMaxLegs = 0
	}
	if n, err := strconv.Atoi(strings.TrimSpace(s.bulkRecEntry.Text)); err == nil && n >= 5 {
		set.BulkRecoverySeconds = n
	} else if set.BulkRecoverySeconds <= 0 {
		set.BulkRecoverySeconds = 60
	}
	set.BulkFallbackMode = store.BulkFallbackLegacyProxy
	set.MultipathEnabled = s.multipathOn.Checked
	preset := s.mpPreset.Selected
	if preset == "" {
		preset = "normal"
	}
	set.MultipathPreset = preset
	set.MultipathWLEmergencyOnly = s.mpWLEmerg.Checked
	set.WLBuiltinConnectEnabled = s.wlBuiltinOn.Checked
	return set, nil
}

func (s *settingsTab) persistSettings(manualReload bool) {
	set, err := s.readForm()
	if err != nil {
		s.statusLabel.SetText("Ошибка: " + err.Error())
		return
	}
	prev := s.lastSaved
	modeOrPort := set.ServiceMode != prev.ServiceMode || set.MixedPort != prev.MixedPort
	mpChanged := set.MultipathEnabled != prev.MultipathEnabled ||
		set.MultipathPreset != prev.MultipathPreset ||
		set.MultipathWLEmergencyOnly != prev.MultipathWLEmergencyOnly ||
		set.WLBuiltinConnectEnabled != prev.WLBuiltinConnectEnabled
	poolChanged := set.AggregationMode != prev.AggregationMode ||
		set.BulkEnabled != prev.BulkEnabled ||
		set.BulkMinHealthyLegs != prev.BulkMinHealthyLegs ||
		set.BulkMaxLegs != prev.BulkMaxLegs ||
		set.BulkRecoverySeconds != prev.BulkRecoverySeconds ||
		set.BulkLBStrategy != prev.BulkLBStrategy

	if _, err := s.app.Config.SaveSettings(s.ctx, set); err != nil {
		s.statusLabel.SetText("Ошибка сохранения: " + err.Error())
		return
	}
	s.lastSaved = set

	connected := false
	if st, err := s.app.Service.Status(s.ctx); err == nil {
		connected = st.IsConnected()
	}

	var parts []string
	parts = append(parts, "Сохранено")
	if set.AggregationMode == store.AggregationModeFlowAggregate && set.BulkEnabled {
		strat := "sticky"
		if set.BulkLBStrategy == store.BulkLBConsistentHash {
			strat = "consistent"
		}
		parts = append(parts, "пул load-balance ("+strat+")")
	} else if set.AggregationMode == store.AggregationModeFlowAggregate {
		parts = append(parts, "пул выкл")
	} else {
		parts = append(parts, "один туннель")
	}
	if set.MultipathEnabled {
		parts = append(parts, "multipath "+set.MultipathPreset)
	}
	if mpChanged || poolChanged {
		parts = append(parts, "→ переподключитесь")
	}
	if modeOrPort {
		if connected {
			if err := s.app.Service.Reload(s.ctx); err != nil {
				parts = append(parts, "Reload mihomo: "+err.Error())
			} else {
				parts = append(parts, "mihomo перезагружен (режим/порт)")
			}
		} else {
			parts = append(parts, "режим/порт — при следующем подключении")
		}
	}
	if manualReload && connected && !modeOrPort {
		if err := s.app.Service.Reload(s.ctx); err != nil {
			parts = append(parts, "Reload: "+err.Error())
		} else {
			parts = append(parts, "Reload OK")
		}
	}
	s.statusLabel.SetText(joinParts(parts))
	if s.pres != nil {
		_ = s.pres.Refresh(s.ctx)
	}
}

func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += ". " + parts[i]
	}
	return out
}

func (s *settingsTab) reloadNow() {
	st, err := s.app.Service.Status(s.ctx)
	if err != nil || !st.IsConnected() {
		dialog.ShowInformation("Reload", "Сейчас нет активного туннеля. Reload нужен после подключения при смене proxy/VPN или порта.", s.w)
		return
	}
	s.persistSettings(true)
}

func (s *settingsTab) stopDaemon() {
	dialog.ShowConfirm("Остановить демон", "Завершить процесс демона? GUI останется открытым.", func(ok bool) {
		if !ok {
			return
		}
		go func() {
			err := platform.StopDaemon(s.ctx, s.app.Layout)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, s.w)
					return
				}
				s.statusLabel.SetText("Демон остановлен")
			})
		}()
	}, s.w)
}

func (s *settingsTab) startDaemon() {
	args := s.opt.DaemonArgs
	if len(args) == 0 {
		args = defaultDaemonArgs(s.opt)
	}
	go func() {
		err := platform.EnsureDaemon(s.ctx, s.app.Layout, args)
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, s.w)
				return
			}
			s.statusLabel.SetText("Демон запущен: " + s.app.Layout.SocketPath())
		})
	}()
}
