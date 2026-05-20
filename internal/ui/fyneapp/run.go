//go:build cgo

package fyneapp

import (
	"context"
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/ui/model"
	"github.com/muhomor/muhomor/internal/ui/presenter"
)

// Options for GUI launch.
type Options struct {
	Layout      paths.Layout
	ServiceMode string
	MixedPort   int
	DaemonArgs  []string
}

// Run starts Fyne simple UI + tray.
func Run(ctx context.Context, opt Options) error {
	api := &apiclient.Client{Dial: apiclient.DialConfig{SocketPath: opt.Layout.SocketPath()}}
	core := appcore.NewApp(opt.Layout, &appcore.DaemonClient{API: api}, &appcore.RemoteConfig{API: api}, nil)
	core.Events = &appcore.DaemonEvents{API: api}

	a := app.NewWithID("com.muhomor.gui")
	a.SetIcon(nil)
	w := a.NewWindow("muhomor")
	w.Resize(fyne.NewSize(420, 320))
	w.SetFixedSize(false)

	statusLabel := widget.NewLabel("Загрузка…")
	activityLabel := widget.NewLabel("")
	profileLabel := widget.NewLabel("")
	portLabel := widget.NewLabel("")
	modeLabel := widget.NewLabel("")

	connectBtn := widget.NewButton("Подключить", nil)
	exportBtn := widget.NewButton("Экспорт лога", nil)
	proxyBtn := widget.NewButton("Режим: Proxy", nil)
	vpnBtn := widget.NewButton("Режим: VPN", nil)

	var pres *presenter.Presenter
	updateUI := func(c model.ConnectionUI, s model.SettingsUI) {
		fyne.Do(func() {
			statusLabel.SetText(fmt.Sprintf("Состояние: %s", c.State))
			if c.Connected {
				profileLabel.SetText(fmt.Sprintf("Профиль: %s\nПрокси: %s", c.ProfileName, c.ProxyName))
				connectBtn.SetText("Отключить")
			} else {
				profileLabel.SetText("")
				connectBtn.SetText("Подключить")
			}
			if c.ActivityText != "" {
				activityLabel.SetText("Активность: " + c.ActivityText)
			} else {
				activityLabel.SetText("")
			}
			portLabel.SetText(fmt.Sprintf("Порт mixed: %d", s.MixedPort))
			modeLabel.SetText("Режим: " + s.ServiceMode)
			if c.ErrorText != "" {
				statusLabel.SetText("Ошибка: " + c.ErrorText)
			}
			connectBtn.Enable()
			if c.Busy {
				connectBtn.Disable()
			}
		})
	}

	pres = presenter.New(core, updateUI)

	connectBtn.OnTapped = func() {
		c, _ := pres.Snapshot()
		connectBtn.Disable()
		go func() {
			var err error
			if c.Connected {
				err = pres.Disconnect(ctx)
			} else {
				err = pres.Connect(ctx)
			}
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, w)
				}
			})
		}()
	}

	exportBtn.OnTapped = func() {
		go func() {
			path, err := pres.ExportLog(ctx)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				dialog.ShowInformation("Лог", path, w)
			})
		}()
	}

	proxyBtn.OnTapped = func() {
		go func() { _ = pres.SetServiceMode(ctx, appcore.ServiceModeProxy) }()
	}
	vpnBtn.OnTapped = func() {
		go func() { _ = pres.SetServiceMode(ctx, appcore.ServiceModeVPN) }()
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle("muhomor — Simple mode", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusLabel,
		activityLabel,
		profileLabel,
		portLabel,
		modeLabel,
		connectBtn,
		container.NewHBox(proxyBtn, vpnBtn),
		exportBtn,
	)
	w.SetContent(container.NewPadded(content))

	setupTray(a, w, pres, ctx)

	w.SetCloseIntercept(func() {
		w.Hide()
	})

	go func() {
		args := opt.DaemonArgs
		if len(args) == 0 {
			args = defaultDaemonArgs(opt)
		}
		if err := pres.Start(ctx, args); err != nil {
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("демон: %w", err), w)
				w.Show()
			})
			return
		}
		fyne.Do(func() { w.Show() })
	}()

	a.Run()
	return nil
}

func defaultDaemonArgs(opt Options) []string {
	var args []string
	if opt.ServiceMode != "" {
		args = append(args, "--service-mode", opt.ServiceMode)
	}
	if opt.MixedPort > 0 {
		args = append(args, "--mixed-port", strconv.Itoa(opt.MixedPort))
	}
	return args
}
