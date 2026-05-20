package appcore

import (
	"context"
	"time"

	"github.com/muhomor/muhomor/internal/apiclient"
)

// ShutdownDaemon stops the muhomor daemon process (not just VPN service).
func (a *App) ShutdownDaemon(ctx context.Context) error {
	if dc, ok := a.Service.(*DaemonClient); ok && dc.API != nil {
		return dc.API.ShutdownDaemon(ctx)
	}
	return nil
}

// WaitDaemonDown polls until daemon HTTP is unreachable.
func WaitDaemonDown(ctx context.Context, api *apiclient.Client, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if api.Reachable(ctx) != nil {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(300 * time.Millisecond):
		}
	}
	return api.Reachable(ctx) != nil
}
