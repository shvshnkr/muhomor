//go:build cgo

package fyneapp

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/muhomor/muhomor/internal/ui/presenter"
)

func setupTray(a fyne.App, w fyne.Window, inner *container.InnerWindow, pres *presenter.Presenter, ctx context.Context) {
	if desk, ok := a.(desktop.App); ok {
		icon := appIcon()
		desk.SetSystemTrayIcon(icon)
		desk.SetSystemTrayWindow(w)
		showItem := fyne.NewMenuItem("Показать", func() {
			w.Show()
			syncWindowLayout(w, inner)
		})
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
		quitItem := fyne.NewMenuItem("Выход", func() { quitApp(a) })
		menu := fyne.NewMenu("muhomor", showItem, startItem, stopItem,
			fyne.NewMenuItemSeparator(), quitItem)
		desk.SetSystemTrayMenu(menu)
	}
}
