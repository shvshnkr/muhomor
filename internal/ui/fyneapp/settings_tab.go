//go:build cgo

package fyneapp

import (
	"context"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/platform"
	"github.com/muhomor/muhomor/internal/ui/presenter"
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
	mixedEntry  *widget.Entry
	modeRadio   *widget.RadioGroup
	multipathOn *widget.Check
	mpPreset    *widget.Select
	mpWLEmerg      *widget.Check
	wlBuiltinOn    *widget.Check
}

func newSettingsTab(w fyne.Window, app *appcore.App, ctx context.Context, opt Options, onBack func()) *settingsTab {
	s := &settingsTab{w: w, app: app, ctx: ctx, opt: opt, onBack: onBack}
	s.statusLabel = widget.NewLabel("")
	s.mixedEntry = widget.NewEntry()
	s.mixedEntry.SetPlaceHolder("2181")
	s.modeRadio = widget.NewRadioGroup([]string{"Proxy (mixed-port)", "VPN (TUN)"}, nil)
	s.multipathOn = widget.NewCheck("Multipath — умный выбор каналов (goodput, не только пинг)", nil)
	s.mpPreset = widget.NewSelect([]string{"low", "normal", "high"}, nil)
	s.mpPreset.SetSelected("normal")
	s.mpWLEmerg = widget.NewCheck("WL только при деградации подписок (multipath)", nil)
	s.mpWLEmerg.SetChecked(true)
	s.wlBuiltinOn = widget.NewCheck("WL builtin trojan — аварийный fallback (выкл = только подписки)", nil)
	s.wlBuiltinOn.SetChecked(false)

	backBtn := widget.NewButton("← Простой режим", func() { s.onBack() })
	saveBtn := widget.NewButton("Сохранить настройки", s.saveSettings)
	reloadBtn := widget.NewButton("Reload mihomo", s.reload)
	daemonStartBtn := widget.NewButton("Запустить демон", s.startDaemon)
	daemonStopBtn := widget.NewButton("Остановить демон", s.stopDaemon)

	s.content = container.NewVBox(
		backBtn,
		widget.NewLabelWithStyle("Настройки", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Mixed port"),
		s.mixedEntry,
		widget.NewLabel("Режим сервиса"),
		s.modeRadio,
		container.NewHBox(saveBtn, reloadBtn),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Multipath", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Агрегация каналов без VPS: ранжирование по пропускной и стабильности. По умолчанию выкл."),
		s.multipathOn,
		widget.NewLabel("Профиль нагрузки"),
		s.mpPreset,
		s.mpWLEmerg,
		s.wlBuiltinOn,
		widget.NewSeparator(),
		widget.NewLabel("Демон"),
		container.NewHBox(daemonStartBtn, daemonStopBtn),
		widget.NewLabel("Остановить демон — завершить процесс muhomor --daemon (не только VPN)."),
		s.statusLabel,
	)
	return s
}

func (s *settingsTab) bindPresenter(pres *presenter.Presenter) {
	s.pres = pres
}

func (s *settingsTab) load(ctx context.Context) {
	set, err := s.app.Config.LoadSettings(ctx)
	if err != nil {
		return
	}
	fyne.Do(func() {
		s.mixedEntry.SetText(strconv.Itoa(set.MixedPort))
		if set.ServiceMode == appcore.ServiceModeVPN {
			s.modeRadio.SetSelected("VPN (TUN)")
		} else {
			s.modeRadio.SetSelected("Proxy (mixed-port)")
		}
		s.multipathOn.SetChecked(set.MultipathEnabled)
		preset := set.MultipathPreset
		if preset == "" {
			preset = "normal"
		}
		s.mpPreset.SetSelected(preset)
		s.mpWLEmerg.SetChecked(set.MultipathWLEmergencyOnly)
		s.wlBuiltinOn.SetChecked(set.WLBuiltinConnectEnabled)
		s.updateMultipathWidgets()
	})
	s.multipathOn.OnChanged = func(bool) { fyne.Do(s.updateMultipathWidgets) }
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

func (s *settingsTab) saveSettings() {
	port, _ := strconv.Atoi(s.mixedEntry.Text)
	set, err := s.app.Config.LoadSettings(s.ctx)
	if err != nil {
		dialog.ShowError(err, s.w)
		return
	}
	set.MixedPort = port
	if s.modeRadio.Selected == "VPN (TUN)" {
		set.ServiceMode = appcore.ServiceModeVPN
		set.TunEnable = true
	} else {
		set.ServiceMode = appcore.ServiceModeProxy
		set.TunEnable = false
	}
	set.MultipathEnabled = s.multipathOn.Checked
	preset := s.mpPreset.Selected
	if preset == "" {
		preset = "normal"
	}
	set.MultipathPreset = preset
	set.MultipathWLEmergencyOnly = s.mpWLEmerg.Checked
	set.WLBuiltinConnectEnabled = s.wlBuiltinOn.Checked
	go func() {
		_, err := s.app.Config.SaveSettings(s.ctx, set)
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, s.w)
				return
			}
			msg := "Сохранено — при подключении нажмите Reload или переподключитесь"
			if set.MultipathEnabled {
				msg += " (multipath включён)"
			}
			s.statusLabel.SetText(msg)
		})
	}()
}

func (s *settingsTab) reload() {
	go func() {
		err := s.app.Service.Reload(s.ctx)
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, s.w)
				return
			}
			s.statusLabel.SetText("Reload OK")
		})
	}()
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
