package pseudogui

import (
	"context"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/ui/console"
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

	// Activity lines render on the Simple TUI status area; avoid spamming stdout in raw mode.
	pres := presenter.New(app, nil)

	if err := pres.Start(ctx, opt.DaemonArgs); err != nil {
		io.Line("демон: " + err.Error())
		io.Line(app.DaemonHint())
		return 1
	}
	return Run(ctx, Config{App: app, IO: io, Pres: pres, DaemonArgs: opt.DaemonArgs})
}
