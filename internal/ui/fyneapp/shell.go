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
	"github.com/muhomor/muhomor/internal/platform"
	"github.com/muhomor/muhomor/internal/ui/model"
	"github.com/muhomor/muhomor/internal/ui/presenter"
)

// Options for GUI launch.
type Options struct {
	Layout      paths.Layout
	ServiceMode string
	MixedPort   int
	RouteQuick  int // -1 = do not pass to daemon
	DaemonArgs  []string
}

// Run starts Fyne UI: simple screen by default; extended tabs on demand.
func Run(ctx context.Context, opt Options) error {
	platform.StaleGUILock(opt.Layout.DataDir)
	unlock, err := platform.AcquireGUILock(opt.Layout.DataDir)
	if err != nil {
		return err
	}
	defer unlock()

	api := &apiclient.Client{Dial: apiclient.DialConfig{SocketPath: opt.Layout.SocketPath()}}
	core := appcore.NewApp(opt.Layout, &appcore.DaemonClient{API: api}, &appcore.RemoteConfig{API: api}, nil)
	core.Groups = &appcore.RemoteGroups{API: api}
	core.Events = &appcore.DaemonEvents{API: api}

	a := app.NewWithID("com.muhomor.gui")
	applyTheme(a)
	frame := newFramedWindow(a, "")
	w := frame.win
	a.Lifecycle().SetOnEnteredForeground(func() {
		syncWindowLayout(w, frame.inner)
	})
	applyAppIcon(a, w)
	w.Resize(fyne.NewSize(windowSimpleW, windowSimpleH))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	var pres *presenter.Presenter
	var showExtended func()

	config := newConfigTab(w, core, ctx, nil)
	route := newRouteTab(w, core, ctx, nil)
	settings := newSettingsTab(w, core, ctx, opt, nil)

	simple := newSimpleTab(w, nil)

	showExtended = func() {
		config.load(ctx)
		route.load(ctx)
		settings.load(ctx)
		frame.setBody(extendedTabs(simple, config, route, settings))
		w.SetFixedSize(false)
		w.Resize(fyne.NewSize(windowExtW, windowExtH))
		frame.setTitle("расширенный режим")
		w.SetTitle("расширенный режим")
		w.CenterOnScreen()
		syncWindowLayout(w, frame.inner)
	}
	showSimple := func() {
		frame.setBody(simple.content)
		w.SetFixedSize(true)
		w.Resize(fyne.NewSize(windowSimpleW, windowSimpleH))
		frame.setTitle("")
		w.SetTitle("")
		w.CenterOnScreen()
		syncWindowLayout(w, frame.inner)
	}
	simple.setFullMode(showExtended)
	config.onBack = showSimple
	route.onBack = showSimple
	settings.onBack = showSimple

	simpleUpdate := simple.makeUpdateCallback()
	pres = presenter.New(core, func(c model.ConnectionUI, s model.SettingsUI) {
		simpleUpdate(c, s)
		settings.applyBulkStatus(c)
	})

	simple.wireConnect(ctx, pres)
	simple.wirePing(ctx, pres)
	simple.wireActions(ctx, pres)
	settings.bindPresenter(pres)

	showSimple()
	syncWindowLayout(w, frame.inner)
	setupTray(a, w, frame.inner, pres, ctx)
	frame.setCloseIntercept(func() { hideToTray(w); nativeHideWindow(w) })

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
			syncWindowLayout(w, frame.inner)
			w.Show()
		})
	}()

	a.Run()
	return nil
}

func extendedTabs(simple *simpleTab, config *configTab, route *routeTab, settings *settingsTab) fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItem("Простой", simple.contentEmbedded),
		container.NewTabItem("Конфигурация", config.content),
		container.NewTabItem("Маршрут", route.content),
		container.NewTabItem("Настройки", settings.content),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return container.NewMax(tabs)
}

func defaultDaemonArgs(opt Options) []string {
	var args []string
	if opt.ServiceMode != "" {
		args = append(args, "--service-mode", opt.ServiceMode)
	}
	if opt.MixedPort > 0 {
		args = append(args, "--mixed-port", strconv.Itoa(opt.MixedPort))
	}
	if opt.RouteQuick >= 0 {
		args = append(args, "--route-quick-profile", strconv.Itoa(opt.RouteQuick))
	}
	return args
}
