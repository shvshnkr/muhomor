package simplemode

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/muhomor/muhomor/internal/reachability"
	"github.com/muhomor/muhomor/internal/selector"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
	"time"
)

// Connector implements SimpleModeConnectCoordinator (Phase 2).
type Connector struct {
	Store      *store.Store
	Selector   *selector.Selector
	Probe      func(context.Context, bool) reachability.Result
	StartFn    func(context.Context, store.Profile, reachability.Result) error
	Bootstrap  bool
	Updater    *subscription.Updater
	Log        *slog.Logger
}

func (c *Connector) Connect(ctx context.Context) error {
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Bootstrap {
		_ = subscription.Bootstrap(ctx, c.Store)
	}
	c.Selector.CancelConnect()
	probe := c.Probe(ctx, true)
	c.Log.Info("reachability", "google", probe.GoogleReachable, "dzen", probe.DzenReachable,
		"whitelist_only", probe.WhitelistOnly(), "event", "H15")
	if !probe.AnyReachable() {
		return fmt.Errorf("network unreachable")
	}
	_ = c.Store.SetKV(ctx, store.KeySimpleModeUseWLPoolOnly, boolKV(probe.WhitelistOnly()))
	_ = c.Store.SetKV(ctx, store.KeyActiveWhitelistRestricted, boolKV(probe.WhitelistOnly()))

	if c.Updater != nil && probe.AnyReachable() {
		budgetCtx, cancel := context.WithTimeout(ctx, connectRefreshBudget(probe.WhitelistOnly()))
		_ = c.Updater.RefreshDue(budgetCtx, true)
		cancel()
	}

	best, res, err := c.Selector.Prepare(ctx, selector.PrepareOpts{
		Owner:         selector.OwnerConnect,
		WhitelistOnly: probe.WhitelistOnly(),
	})
	if err != nil {
		return err
	}
	switch res {
	case selector.ResultNoProfiles:
		return fmt.Errorf("no profiles; run bootstrap or --import-uri")
	case selector.ResultAllDead:
		return fmt.Errorf("all servers failed probes")
	}
	return c.StartFn(ctx, best, probe)
}

func connectRefreshBudget(wl bool) time.Duration {
	if wl {
		return 8 * time.Second
	}
	return 2800 * time.Millisecond
}

func boolKV(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
