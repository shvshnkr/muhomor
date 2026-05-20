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

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/ui/presenter"
)

// Options for GUI launch.
type Options struct {
	Layout      paths.Layout
	ServiceMode string
	MixedPort   int
	DaemonArgs  []string
}

// Run starts Fyne UI (simple + full mode) + tray.
func Run(ctx context.Context, opt Options) error {
	api := &apiclient.Client{Dial: apiclient.DialConfig{SocketPath: opt.Layout.SocketPath()}}
	core := appcore.NewApp(opt.Layout, &appcore.DaemonClient{API: api}, &appcore.RemoteConfig{API: api}, nil)
	core.Groups = &appcore.RemoteGroups{API: api}
	core.Events = &appcore.DaemonEvents{API: api}

	a := app.NewWithID("com.muhomor.gui")
	w := a.NewWindow("muhomor")
	w.Resize(fyne.NewSize(720, 520))

	var pres *presenter.Presenter
	var tabs *container.AppTabs

	goSimple := func() {
		if tabs != nil {
			tabs.SelectIndex(0)
		}
	}

	simple := newSimpleTab(w, func() {
		if tabs != nil {
			tabs.SelectIndex(1)
		}
	})
	config := newConfigTab(w, core, ctx, goSimple)
	route := newRouteTab(w, core, ctx, goSimple)
	settings := newSettingsTab(w, core, ctx, opt, goSimple)

	tabs = container.NewAppTabs(
		container.NewTabItem("Простой", container.NewPadded(simple.content)),
		container.NewTabItem("Конфигурация", container.NewPadded(config.content)),
		container.NewTabItem("Маршрут", container.NewPadded(route.content)),
		container.NewTabItem("Настройки", container.NewPadded(settings.content)),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	updateUI := simple.makeUpdateCallback()
	pres = presenter.New(core, updateUI)

	simple.wireConnect(ctx, pres)
	simple.wireActions(ctx, pres)
	settings.bindPresenter(pres)

	w.SetContent(tabs)
	setupTray(a, w, pres, ctx)
	w.SetCloseIntercept(func() { w.Hide() })

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
		fyne.Do(func() {
			config.load(ctx)
			route.load(ctx)
			settings.load(ctx)
			w.Show()
		})
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
