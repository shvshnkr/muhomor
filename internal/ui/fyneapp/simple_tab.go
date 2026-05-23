//go:build cgo

package fyneapp

import (
	"context"
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/muhomor/muhomor/internal/ui/model"
	"github.com/muhomor/muhomor/internal/ui/presenter"
)

type simpleTab struct {
	content       fyne.CanvasObject
	fullMode      func()
	statusDot     *canvas.Circle
	statusLabel   *widget.Label
	activityLabel *widget.Label
	probeLabel    *widget.Label
	pingLabel     *widget.Label
	profileLabel  *widget.Label
	detailCard    *fyne.Container
	connectBtn    *widget.Button
	pingBtn       *widget.Button
	exportBtn     *widget.Button
	w             fyne.Window
}

func newSimpleTab(w fyne.Window, onFullMode func()) *simpleTab {
	t := &simpleTab{w: w, fullMode: onFullMode}
	t.statusLabel = widget.NewLabelWithStyle("Загрузка…", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	t.activityLabel = widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{})
	t.activityLabel.Importance = widget.LowImportance
	t.probeLabel = widget.NewLabel("")
	t.probeLabel.Importance = widget.LowImportance
	t.probeLabel.Wrapping = fyne.TextWrapWord
	t.pingLabel = widget.NewLabel("")
	t.pingLabel.Importance = widget.LowImportance
	t.profileLabel = widget.NewLabel("")
	t.profileLabel.Wrapping = fyne.TextWrapWord

	dot := statusDot(colorMuted)
	t.statusDot = dot
	statusRow := centeredStatusBlock(dot, t.statusLabel, t.activityLabel)

	t.connectBtn = widget.NewButton("Подключить", nil)
	t.connectBtn.Importance = widget.HighImportance
	t.pingBtn = widget.NewButton("Ping", nil)
	t.pingBtn.Importance = widget.MediumImportance
	applyTooltip(t.pingBtn, "Проверка задержки. При активном пуле load-balance — ping всех ног.")
	t.exportBtn = widget.NewButton("Экспорт лога", nil)
	t.exportBtn.Importance = widget.MediumImportance

	fullBtn := widget.NewButton("Расширенный режим", func() {
		if t.fullMode != nil {
			t.fullMode()
		}
	})
	fullBtn.Importance = widget.LowImportance

	t.detailCard = surfaceCard(container.NewVBox(t.profileLabel, t.pingLabel, t.probeLabel), 0).(*fyne.Container)
	t.detailCard.Hide()

	statusCard := surfaceCard(statusRow, 0)

	header := sectionHeader("muhomor", "Подключение с умным выбором канала")

	t.content = container.NewVBox(
		header,
		vSpacer(4),
		statusCard,
		t.detailCard,
		vSpacer(8),
		container.NewHBox(t.connectBtn, t.pingBtn),
		container.NewBorder(nil, nil, t.exportBtn, fullBtn, nil),
	)
	return t
}

func (t *simpleTab) setFullMode(fn func()) {
	t.fullMode = fn
}

func (t *simpleTab) setStatusDot(c color.Color) {
	if t.statusDot != nil {
		t.statusDot.FillColor = c
		t.statusDot.Refresh()
	}
}

func (t *simpleTab) wireConnect(ctx context.Context, pres *presenter.Presenter) {
	t.connectBtn.OnTapped = func() {
		c, _ := pres.Snapshot()
		switch {
		case c.Busy && !c.Connected:
			go pres.AbortConnect(ctx)
			return
		case c.Connected:
			t.connectBtn.Disable()
			go func() {
				err := pres.Disconnect(ctx)
				fyne.Do(func() {
					c2, _ := pres.Snapshot()
					t.syncConnectButton(c2)
					if err != nil {
						dialog.ShowInformation("Отключение", model.FriendlyConnectError(err), t.w)
					}
				})
			}()
		case c.Busy:
			return
		default:
			t.connectBtn.Disable()
			go func() {
				err := pres.Connect(ctx)
				fyne.Do(func() {
					if err != nil {
						dialog.ShowInformation("Подключение", model.FriendlyConnectError(err), t.w)
					}
				})
			}()
		}
	}
}

func (t *simpleTab) syncConnectButton(c model.ConnectionUI) {
	if c.Busy {
		t.connectBtn.SetText("Отменить")
		t.connectBtn.Enable()
		return
	}
	if c.Connected {
		t.connectBtn.SetText("Отключить")
	} else {
		t.connectBtn.SetText("Подключить")
	}
	t.connectBtn.Enable()
}

func (t *simpleTab) wirePing(ctx context.Context, pres *presenter.Presenter) {
	t.pingBtn.OnTapped = func() {
		c, _ := pres.Snapshot()
		if !c.Connected {
			dialog.ShowInformation("Ping", "Сначала подключитесь.", t.w)
			return
		}
		bulk := len(c.BulkMembers) > 0
		t.pingBtn.Disable()
		go func() {
			if bulk {
				resp, err := pres.BulkPingAll(ctx)
				fyne.Do(func() {
					t.pingBtn.Enable()
					if err != nil {
						dialog.ShowError(err, t.w)
						return
					}
					if resp.Error != "" {
						dialog.ShowInformation("Ping пула", resp.Error, t.w)
						return
					}
					dialog.ShowInformation("Ping пула", fmt.Sprintf("%d/%d живых", resp.OK, resp.Total), t.w)
				})
				return
			}
			resp, err := pres.Ping(ctx)
			fyne.Do(func() {
				t.pingBtn.Enable()
				if err != nil {
					dialog.ShowError(err, t.w)
					return
				}
				if resp.Error != "" {
					dialog.ShowInformation("Ping", resp.Error, t.w)
				} else if resp.DelayMs > 0 {
					dialog.ShowInformation("Ping", fmt.Sprintf("%d ms", resp.DelayMs), t.w)
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

func (t *simpleTab) applyVisualState(c model.ConnectionUI) {
	var dot color.Color
	var btnImp widget.Importance
	switch {
	case c.ErrorText != "":
		dot = colorError
		btnImp = widget.HighImportance
	case c.Busy:
		dot = colorWarn
		btnImp = widget.HighImportance
	case c.Connected:
		dot = colorSuccess
		btnImp = widget.DangerImportance
	default:
		dot = colorMuted
		btnImp = widget.HighImportance
	}
	t.setStatusDot(dot)
	t.connectBtn.Importance = btnImp
}

func (t *simpleTab) makeUpdateCallback() func(model.ConnectionUI, model.SettingsUI) {
	return func(c model.ConnectionUI, s model.SettingsUI) {
		_ = s
		fyne.Do(func() {
			t.applyVisualState(c)

			if c.ErrorText != "" {
				t.statusLabel.SetText("Ошибка")
				t.activityLabel.SetText(c.ErrorText)
			} else if c.Busy && c.ActivityText != "" {
				t.statusLabel.SetText("Подключение…")
				t.activityLabel.SetText(c.ActivityText)
			} else if c.Busy {
				t.statusLabel.SetText("Подключение…")
				if c.ActivityText != "" {
					t.activityLabel.SetText(c.ActivityText)
				} else {
					t.activityLabel.SetText("")
				}
			} else if c.Connected {
				t.statusLabel.SetText("Подключено")
				if c.ActivityText != "" {
					t.activityLabel.SetText(c.ActivityText)
				} else {
					t.activityLabel.SetText("")
				}
			} else {
				t.statusLabel.SetText("Отключено")
				if c.ActivityText != "" {
					t.activityLabel.SetText(c.ActivityText)
				} else {
					t.activityLabel.SetText("")
				}
			}

			if c.Connected {
				t.profileLabel.SetText(fmt.Sprintf("Профиль: %s\nПрокси: %s", c.ProfileName, c.ProxyName))
			} else {
				t.profileLabel.SetText("")
			}
			t.syncConnectButton(c)
			if c.Busy {
				t.connectBtn.Importance = widget.MediumImportance
			}

			if c.LastPingMs > 0 {
				t.pingLabel.SetText(fmt.Sprintf("Пинг: %d ms", c.LastPingMs))
			} else if c.LastPingError != "" {
				t.pingLabel.SetText("Пинг: " + c.LastPingError)
			} else {
				t.pingLabel.SetText("")
			}

			probeLine := c.ProbeText
			if c.MultipathText != "" {
				if probeLine != "" {
					probeLine += "\n" + c.MultipathText
				} else {
					probeLine = c.MultipathText
				}
			}
			if table := model.FormatBulkMembersTable(c.BulkMembers); table != "" {
				if probeLine != "" {
					probeLine += "\n" + table
				} else {
					probeLine = table
				}
			}
			if probeLine != "" && c.ErrorText == "" {
				t.probeLabel.SetText(probeLine)
			} else {
				t.probeLabel.SetText("")
			}

			hasBulk := len(c.BulkMembers) > 0
			if c.Connected && hasBulk {
				t.pingBtn.SetText("Ping всех")
				t.pingBtn.Enable()
			} else if c.Connected {
				t.pingBtn.SetText("Ping")
				t.pingBtn.Enable()
			} else {
				t.pingBtn.SetText("Ping")
				t.pingBtn.Disable()
			}

			showDetail := (c.Connected && t.profileLabel.Text != "") || t.probeLabel.Text != "" || t.pingLabel.Text != "" || c.Busy
			if showDetail {
				t.detailCard.Show()
			} else {
				t.detailCard.Hide()
			}

		})
	}
}
