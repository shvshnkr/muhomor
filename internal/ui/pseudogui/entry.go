package pseudogui

import (
	"context"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/ui/console"
)

// Start runs pseudo-GUI using daemon HTTP only (Phase 4.0).
func Start(ctx context.Context, layout paths.Layout) int {
	console.PrepareConsole()
	io := console.StdIO()

	api := &apiclient.Client{Dial: apiclient.DialConfig{SocketPath: layout.SocketPath()}}
	svc := &appcore.DaemonClient{API: api}
	cfg := &appcore.RemoteConfig{API: api}
	app := appcore.NewApp(layout, svc, cfg, io)
	app.Events = &appcore.DaemonEvents{API: api}

	return Run(ctx, Config{App: app, IO: io})
}
