//go:build cgo

package fyneapp

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/ui/model"
	"github.com/muhomor/muhomor/internal/ui/presenter"
)

type simpleTab struct {
	content        *fyne.Container
	statusLabel    *widget.Label
	activityLabel  *widget.Label
	profileLabel   *widget.Label
	portLabel      *widget.Label
	modeLabel      *widget.Label
	connectBtn     *widget.Button
	exportBtn      *widget.Button
	proxyBtn       *widget.Button
	vpnBtn         *widget.Button
	delayBtn       *widget.Button
	w              fyne.Window
	pres           *presenter.Presenter
}

func newSimpleTab(w fyne.Window, onFullMode func()) *simpleTab {
	t := &simpleTab{w: w}
	fullBtn := widget.NewButton("Полный режим →", onFullMode)
	t.statusLabel = widget.NewLabel("Загрузка…")
	t.activityLabel = widget.NewLabel("")
	t.profileLabel = widget.NewLabel("")
	t.portLabel = widget.NewLabel("")
	t.modeLabel = widget.NewLabel("")
	t.connectBtn = widget.NewButton("Подключить", nil)
	t.exportBtn = widget.NewButton("Экспорт лога", nil)
	t.proxyBtn = widget.NewButton("Режим: Proxy", nil)
	t.vpnBtn = widget.NewButton("Режим: VPN", nil)
	t.delayBtn = widget.NewButton("Задержка текущего", nil)
	t.delayBtn.Disable()
	t.content = container.NewVBox(
		widget.NewLabelWithStyle("muhomor — простой режим", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		t.statusLabel,
		t.activityLabel,
		t.profileLabel,
		t.portLabel,
		t.modeLabel,
		t.connectBtn,
		container.NewHBox(t.proxyBtn, t.vpnBtn),
		t.delayBtn,
		t.exportBtn,
		fullBtn,
	)
	return t
}

func (t *simpleTab) wireConnect(ctx context.Context, pres *presenter.Presenter) {
	t.connectBtn.OnTapped = func() {
		c, _ := pres.Snapshot()
		t.connectBtn.Disable()
		go func() {
			var err error
			if c.Connected {
				err = pres.Disconnect(ctx)
			} else {
				err = pres.Connect(ctx)
			}
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, t.w)
				}
			})
		}()
	}
}

func (t *simpleTab) makeUpdateCallback() func(model.ConnectionUI, model.SettingsUI) {
	return func(c model.ConnectionUI, s model.SettingsUI) {
		fyne.Do(func() {
			if c.ErrorText != "" {
				t.statusLabel.SetText("Ошибка: " + c.ErrorText)
			} else if c.Busy && c.ActivityText != "" {
				t.statusLabel.SetText(c.ActivityText)
			} else if c.Busy {
				t.statusLabel.SetText("Подключение…")
			} else {
				t.statusLabel.SetText(fmt.Sprintf("Состояние: %s", c.State))
			}
			if c.Connected {
				t.profileLabel.SetText(fmt.Sprintf("Профиль: %s\nПрокси: %s", c.ProfileName, c.ProxyName))
				t.connectBtn.SetText("Отключить")
				t.delayBtn.Enable()
			} else {
				t.profileLabel.SetText("")
				t.connectBtn.SetText("Подключить")
				t.delayBtn.Disable()
			}
			if c.ActivityText != "" && c.ErrorText == "" {
				t.activityLabel.SetText(c.ActivityText)
			} else {
				t.activityLabel.SetText("")
			}
			t.portLabel.SetText(fmt.Sprintf("Порт mixed: %d", s.MixedPort))
			mode := "Proxy (mixed-port, без TUN)"
			if s.ServiceMode == appcore.ServiceModeVPN {
				mode = "VPN (TUN + mihomo)"
			}
			t.modeLabel.SetText("Режим: " + mode)
			t.connectBtn.Enable()
			if c.Busy {
				t.connectBtn.Disable()
			}
		})
	}
}

func (t *simpleTab) wireActions(ctx context.Context, pres *presenter.Presenter) {
	t.pres = pres
	t.delayBtn.OnTapped = func() {
		c, _ := pres.Snapshot()
		if c.ProfileID == 0 || pres.App.Groups == nil {
			return
		}
		t.delayBtn.Disable()
		go func() {
			res, err := pres.App.Groups.TestProfileDelay(ctx, c.ProfileID)
			fyne.Do(func() {
				t.delayBtn.Enable()
				if err != nil {
					dialog.ShowError(err, t.w)
					return
				}
				if res.Error != "" {
					dialog.ShowInformation("Задержка", res.Error, t.w)
				} else {
					dialog.ShowInformation("Задержка", fmt.Sprintf("%d ms", res.DelayMs), t.w)
				}
			})
		}()
	}
	t.exportBtn.OnTapped = func() {
		go func() {
			path, err := pres.ExportLog(ctx)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, t.w)
					return
				}
				dialog.ShowInformation("Лог", path, t.w)
			})
		}()
	}
	t.proxyBtn.OnTapped = func() {
		go func() { _ = pres.SetServiceMode(ctx, appcore.ServiceModeProxy) }()
	}
	t.vpnBtn.OnTapped = func() {
		go func() { _ = pres.SetServiceMode(ctx, appcore.ServiceModeVPN) }()
	}
}
