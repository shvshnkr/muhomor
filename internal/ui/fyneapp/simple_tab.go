//go:build cgo

package fyneapp

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/muhomor/muhomor/internal/ui/model"
	"github.com/muhomor/muhomor/internal/ui/presenter"
)

type simpleTab struct {
	content       *fyne.Container
	fullMode      func()
	statusLabel   *widget.Label
	activityLabel *widget.Label
	probeLabel    *widget.Label
	profileLabel  *widget.Label
	connectBtn    *widget.Button
	exportBtn     *widget.Button
	w             fyne.Window
}

func newSimpleTab(w fyne.Window, onFullMode func()) *simpleTab {
	t := &simpleTab{w: w, fullMode: onFullMode}
	t.statusLabel = widget.NewLabel("Загрузка…")
	t.activityLabel = widget.NewLabel("")
	t.probeLabel = widget.NewLabel("")
	t.profileLabel = widget.NewLabel("")
	t.connectBtn = widget.NewButton("Подключить", nil)
	t.exportBtn = widget.NewButton("Экспорт лога", nil)
	fullBtn := widget.NewButton("Расширенный режим →", func() {
		if t.fullMode != nil {
			t.fullMode()
		}
	})

	t.content = container.NewVBox(
		widget.NewLabelWithStyle("muhomor", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		t.statusLabel,
		t.activityLabel,
		t.probeLabel,
		t.profileLabel,
		t.connectBtn,
		t.exportBtn,
		fullBtn,
	)
	return t
}

func (t *simpleTab) setFullMode(fn func()) {
	t.fullMode = fn
}

func (t *simpleTab) wireConnect(ctx context.Context, pres *presenter.Presenter) {
	t.connectBtn.OnTapped = func() {
		c, _ := pres.Snapshot()
		go func() {
			var err error
			if c.Busy || c.Connected {
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

func (t *simpleTab) wireActions(ctx context.Context, pres *presenter.Presenter) {
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
}

func (t *simpleTab) makeUpdateCallback() func(model.ConnectionUI, model.SettingsUI) {
	return func(c model.ConnectionUI, s model.SettingsUI) {
		_ = s
		fyne.Do(func() {
			if c.ErrorText != "" {
				t.statusLabel.SetText("Ошибка: " + c.ErrorText)
			} else if c.Busy && c.ActivityText != "" {
				t.statusLabel.SetText(c.ActivityText)
			} else if c.Busy {
				if c.ActivityText != "" {
					t.statusLabel.SetText(c.ActivityText)
				} else {
					t.statusLabel.SetText("Подключение…")
				}
			} else {
				t.statusLabel.SetText(fmt.Sprintf("Состояние: %s", c.State))
			}
			if c.Busy {
				t.connectBtn.SetText("Отменить")
			} else if c.Connected {
				t.profileLabel.SetText(fmt.Sprintf("Профиль: %s\nПрокси: %s", c.ProfileName, c.ProxyName))
				t.connectBtn.SetText("Отключить")
			} else {
				t.profileLabel.SetText("")
				t.connectBtn.SetText("Подключить")
			}
			if c.ActivityText != "" && c.ErrorText == "" {
				t.activityLabel.SetText(c.ActivityText)
			} else {
				t.activityLabel.SetText("")
			}
			probeLine := c.ProbeText
			if c.MultipathText != "" {
				if probeLine != "" {
					probeLine += "\n" + c.MultipathText
				} else {
					probeLine = c.MultipathText
				}
			}
			if probeLine != "" && c.ErrorText == "" {
				t.probeLabel.SetText(probeLine)
			} else {
				t.probeLabel.SetText("")
			}
			t.connectBtn.Enable()
		})
	}
}
