package pseudogui

import (
	"context"
	"sync"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/ui/console"
	"github.com/muhomor/muhomor/internal/ui/model"
	"github.com/muhomor/muhomor/internal/ui/presenter"
)

// Start runs pseudo-GUI (daemon HTTP + presenter, parity with Fyne simple UI).
func Start(ctx context.Context, layout paths.Layout, opt Options) int {
	console.PrepareConsole()
	io := console.StdIO()

	api := &apiclient.Client{Dial: apiclient.DialConfig{SocketPath: layout.SocketPath()}}
	svc := &appcore.DaemonClient{API: api}
	cfg := &appcore.RemoteConfig{API: api}
	app := appcore.NewApp(layout, svc, cfg, io)
	app.Groups = &appcore.RemoteGroups{API: api}
	app.Events = &appcore.DaemonEvents{API: api}

	var actMu sync.Mutex
	var lastActivity string
	pres := presenter.New(app, func(c model.ConnectionUI, s model.SettingsUI) {
		actMu.Lock()
		defer actMu.Unlock()
		if c.Busy && c.ActivityText != "" && c.ActivityText != lastActivity {
			io.Line("  » " + c.ActivityText)
			lastActivity = c.ActivityText
		}
	})

	if err := pres.Start(ctx, opt.DaemonArgs); err != nil {
		io.Line("демон: " + err.Error())
		io.Line(app.DaemonHint())
		return 1
	}
	c, s := pres.Snapshot()
	printStatusBanner(io, c, s)

	return Run(ctx, Config{App: app, IO: io, Pres: pres, DaemonArgs: opt.DaemonArgs})
}
