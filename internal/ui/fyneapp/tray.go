//go:build cgo

package fyneapp

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/ui/presenter"
)

func setupTray(a fyne.App, w fyne.Window, pres *presenter.Presenter, ctx context.Context) {
	if desk, ok := a.(desktop.App); ok {
		showItem := fyne.NewMenuItem("Показать", func() { w.Show() })
		startItem := fyne.NewMenuItem("Подключить", func() {
			go func() { _ = pres.Connect(ctx) }()
		})
		stopItem := fyne.NewMenuItem("Остановить", func() {
			go func() { _ = pres.Disconnect(ctx) }()
		})
		proxyItem := fyne.NewMenuItem("Proxy", func() {
			go func() { _ = pres.SetServiceMode(ctx, appcore.ServiceModeProxy) }()
		})
		vpnItem := fyne.NewMenuItem("VPN", func() {
			go func() { _ = pres.SetServiceMode(ctx, appcore.ServiceModeVPN) }()
		})
		quitItem := fyne.NewMenuItem("Выход", func() { a.Quit() })
		menu := fyne.NewMenu("muhomor", showItem, startItem, stopItem,
			fyne.NewMenuItemSeparator(), proxyItem, vpnItem,
			fyne.NewMenuItemSeparator(), quitItem)
		desk.SetSystemTrayMenu(menu)
		if icon := a.Icon(); icon != nil {
			desk.SetSystemTrayIcon(icon)
		}
	}
}
