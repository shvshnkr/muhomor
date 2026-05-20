package pseudogui

import (
	"context"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/ui/console"
)

// Start opens DB, wires appcore, runs menu. Caller should call layout.Ensure() first.
func Start(ctx context.Context, layout paths.Layout) int {
	console.PrepareConsole()
	io := console.StdIO()

	st, err := store.Open(layout.DBPath())
	if err != nil {
		io.Line("store: " + err.Error())
		return 1
	}
	defer st.Close()

	api := &apiclient.Client{Dial: apiclient.DialConfig{SocketPath: layout.SocketPath()}}
	svc := &appcore.DaemonClient{API: api}
	cfg := &appcore.LocalStoreRepo{Store: st}
	app := appcore.NewApp(layout, svc, cfg, io)

	return Run(ctx, Config{App: app, IO: io})
}
