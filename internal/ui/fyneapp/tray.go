//go:build cgo

package fyneapp

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"

	"github.com/muhomor/muhomor/internal/ui/presenter"
)

func setupTray(a fyne.App, w fyne.Window, pres *presenter.Presenter, ctx context.Context) {
	if desk, ok := a.(desktop.App); ok {
		showItem := fyne.NewMenuItem("Показать", func() { w.Show() })
		startItem := fyne.NewMenuItem("Подключить", func() {
			go func() { _ = pres.Connect(ctx) }()
		})
		stopItem := fyne.NewMenuItem("Остановить", func() {
			go func() {
				c, _ := pres.Snapshot()
				if c.Busy && !c.Connected {
					pres.AbortConnect(ctx)
					return
				}
				_ = pres.Disconnect(ctx)
			}()
		})
		quitItem := fyne.NewMenuItem(lang.L("Quit"), nil)
		quitItem.IsQuit = true
		menu := fyne.NewMenu("muhomor", showItem, startItem, stopItem,
			fyne.NewMenuItemSeparator(), quitItem)
		desk.SetSystemTrayMenu(menu)
		if icon := a.Icon(); icon != nil {
			desk.SetSystemTrayIcon(icon)
		}
	}
}
