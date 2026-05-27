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
	content         fyne.CanvasObject
	contentEmbedded fyne.CanvasObject
	fullMode        func()
	statusDot       *canvas.Circle
	statusLabel     *widget.Label
	activityLabel   *widget.Label
	probeLabel      *widget.Label
	profileLabel    *widget.Label
	pingResultLabel *widget.Label
	pingBusy        bool
	connectBtn      *widget.Button
	pingBtn         *widget.Button
	exportBtn       *widget.Button
	w               fyne.Window
}

func newSimpleTab(w fyne.Window, onFullMode func()) *simpleTab {
	t := &simpleTab{w: w, fullMode: onFullMode}
	t.statusLabel = displayLabel("Загрузка…")
	t.activityLabel = captionLabel("")
	t.activityLabel.Wrapping = fyne.TextWrapWord
	t.probeLabel = captionLabel("")
	t.probeLabel.Wrapping = fyne.TextWrapWord
	t.profileLabel = bodyLabel("")
	t.profileLabel.Wrapping = fyne.TextWrapWord
	t.pingResultLabel = widget.NewLabel("—")
	styleStat(t.pingResultLabel)

	dot := statusDot(colorMuted)
	t.statusDot = dot

	t.connectBtn = widget.NewButton("Подключить", nil)
	t.connectBtn.Importance = widget.HighImportance
	t.pingBtn = secondaryButton("Ping", nil)
	applyTooltip(t.pingBtn, "Проверка задержки. При активном пуле load-balance — ping всех ног.")
	t.exportBtn = secondaryButton("Экспорт лога", nil)

	fullBtn := secondaryButton("Расширенный режим", func() {
		if t.fullMode != nil {
			t.fullMode()
		}
	})

	brand := captionLabel("muhomor")
	brand.Importance = widget.LowImportance

	probeScroll := container.NewScroll(container.NewVBox(t.probeLabel))
	probeScroll.SetMinSize(fyne.NewSize(0, 72))

	meta := container.NewVBox(
		t.profileLabel,
		probeScroll,
	)

	heroInner := container.NewVBox(
		statusHeroRow(dot, t.statusLabel, t.activityLabel),
		subtleDivider(),
		vSpace(space2),
		meta,
	)
	hero := heroConnectCard(heroInner)

	connectWrap := connectButtonRow(t.connectBtn)

	middle := container.NewVBox(
		vSpace(space2),
		hero,
		vSpace(space4),
		connectWrap,
		vSpace(space3),
		container.NewBorder(nil, nil, captionLabel("Задержка"),
			container.NewHBox(t.pingBtn, t.pingResultLabel), nil),
	)

	footer := container.NewBorder(nil, nil, t.exportBtn, fullBtn, nil)

	body := simpleBodyCompact(
		container.NewVBox(brand, vSpace(space2)),
		middle,
		container.NewVBox(vSpace(space3), footer),
	)
	t.content = simplePadded(body)
	t.contentEmbedded = simplePadded(body)
	return t
}

func (t *simpleTab) setFullMode(fn func()) {
	t.fullMode = fn
}

func (t *simpleTab) setStatusDot(c color.Color) {
	if t.statusDot == nil {
		return
	}
	t.statusDot.FillColor = c
	t.statusDot.StrokeColor = color.Transparent
	t.statusDot.StrokeWidth = 0
	t.statusDot.Refresh()
}

func (t *simpleTab) setStatusDotConnected(c color.Color) {
	if t.statusDot == nil {
		return
	}
	t.statusDot.FillColor = c
	t.statusDot.StrokeColor = accentStroke()
	t.statusDot.StrokeWidth = 2
	t.statusDot.Refresh()
}

func (t *simpleTab) setPingInline(text string, errStyle bool) {
	if t.pingResultLabel == nil {
		return
	}
	t.pingResultLabel.SetText(text)
	t.pingResultLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	if errStyle {
		t.pingResultLabel.Importance = widget.DangerImportance
	} else if text != "" && text != "—" && text != "Проверка…" {
		t.pingResultLabel.Importance = widget.SuccessImportance
	} else {
		t.pingResultLabel.Importance = widget.MediumImportance
	}
}

func (t *simpleTab) syncPingFromConnection(c model.ConnectionUI, connecting bool) {
	if t.pingBusy {
		return
	}
	if c.LastPingMs > 0 && (c.Connected || connecting) {
		t.setPingInline(fmt.Sprintf("%d ms", c.LastPingMs), false)
		return
	}
	if c.LastPingError != "" && (c.Connected || connecting) {
		t.setPingInline(c.LastPingError, true)
		return
	}
	if c.Connected || connecting {
		t.setPingInline("…", false)
		return
	}
	t.setPingInline("—", false)
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
			t.setPingInline("Нужно подключение", false)
			return
		}
		bulk := len(c.BulkMembers) > 0
		t.pingBusy = true
		t.pingBtn.Disable()
		t.setPingInline("Проверка…", false)
		go func() {
			if bulk {
				resp, err := pres.BulkPingAll(ctx)
				fyne.Do(func() {
					t.pingBusy = false
					t.pingBtn.Enable()
					if err != nil {
						t.setPingInline(err.Error(), true)
						return
					}
					if resp.Error != "" {
						t.setPingInline(resp.Error, true)
						return
					}
					t.setPingInline(fmt.Sprintf("%d/%d живых", resp.OK, resp.Total), false)
				})
				return
			}
			resp, err := pres.Ping(ctx)
			fyne.Do(func() {
				t.pingBusy = false
				t.pingBtn.Enable()
				if err != nil {
					t.setPingInline(err.Error(), true)
					return
				}
				if resp.Error != "" {
					t.setPingInline(resp.Error, true)
					return
				}
				if resp.DelayMs > 0 {
					t.setPingInline(fmt.Sprintf("%d ms", resp.DelayMs), false)
				} else {
					t.setPingInline("OK", false)
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
	var connected bool
	connecting := uiConnecting(c)
	switch {
	case c.ErrorText != "":
		dot = colorError
		btnImp = widget.HighImportance
	case connecting:
		dot = colorWarn
		btnImp = widget.HighImportance
	case c.Connected:
		dot = colorSuccess
		btnImp = widget.DangerImportance
		connected = true
	default:
		dot = colorMuted
		btnImp = widget.HighImportance
	}
	if connected {
		t.setStatusDotConnected(dot)
	} else {
		t.setStatusDot(dot)
	}
	t.connectBtn.Importance = btnImp
}

func (t *simpleTab) makeUpdateCallback() func(model.ConnectionUI, model.SettingsUI) {
	return func(c model.ConnectionUI, s model.SettingsUI) {
		_ = s
		fyne.Do(func() {
			connecting := uiConnecting(c)
			t.applyVisualState(c)

			switch {
			case c.ErrorText != "":
				t.statusLabel.SetText("Ошибка")
				t.activityLabel.SetText(c.ErrorText)
			case c.Connected:
				t.statusLabel.SetText("Подключено")
				if c.ActivityText != "" {
					t.activityLabel.SetText(c.ActivityText)
				} else {
					t.activityLabel.SetText("")
				}
			case connecting:
				t.statusLabel.SetText("Подключение…")
				if c.ActivityText != "" {
					t.activityLabel.SetText(c.ActivityText)
				} else {
					t.activityLabel.SetText("")
				}
			default:
				t.statusLabel.SetText("Отключено")
				if c.ActivityText != "" && !isStartupActivity(c.ActivityText) {
					t.activityLabel.SetText(c.ActivityText)
				} else {
					t.activityLabel.SetText("")
				}
			}

			if c.Connected && c.ProfileName != "" {
				t.profileLabel.SetText(fmt.Sprintf("Профиль: %s", c.ProfileName))
			} else {
				t.profileLabel.SetText("")
			}
			t.syncConnectButton(c)
			if c.Busy {
				t.connectBtn.Importance = widget.MediumImportance
			}

			t.syncPingFromConnection(c, connecting)

			probeLine := c.ProbeText
			if c.StandbyText != "" {
				if probeLine != "" {
					probeLine += "\n" + c.StandbyText
				} else {
					probeLine = c.StandbyText
				}
			}
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
		})
	}
}
