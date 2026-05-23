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
	Activity   func(context.Context, string)
}

func (c *Connector) Connect(ctx context.Context) error {
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Bootstrap {
		_ = subscription.Bootstrap(ctx, c.Store)
	}
	if n, err := subscription.RepairTruncatedURIs(ctx, c.Store); err != nil {
		return err
	} else if n > 0 && c.Log != nil {
		c.Log.Info("repaired truncated subscription URIs", "count", n, "event", "uri-repair")
	}
	if c.Activity != nil {
		c.Activity(ctx, "Проверка сети…")
	}
	probe := c.Probe(ctx, true)
	c.Log.Info("reachability", "google", probe.GoogleReachable, "dzen", probe.DzenReachable,
		"whitelist_only", probe.WhitelistOnly(), "event", "H15")
	if !probe.AnyReachable() {
		return fmt.Errorf("network unreachable")
	}
	_ = c.Store.SetKV(ctx, store.KeySimpleModeUseWLPoolOnly, boolKV(probe.WhitelistOnly()))
	_ = c.Store.SetKV(ctx, store.KeyActiveWhitelistRestricted, boolKV(probe.WhitelistOnly()))

	if probe.WhitelistOnly() {
		_ = subscription.BootstrapWhiteBoltWL(ctx, c.Store)
	}

	// Subscription refresh on connect blocked the DB (SQLITE_BUSY) and added seconds; scheduler refreshes in background.
	if c.Updater != nil && probe.AnyReachable() && c.Log != nil {
		c.Log.Debug("connect: subscription refresh deferred to scheduler", "wl_only", probe.WhitelistOnly(), "event", "H29-connect")
	}

	opts := selector.PrepareOpts{
		Owner:         selector.OwnerConnect,
		WhitelistOnly: probe.WhitelistOnly(),
	}
	best, res, err := c.Selector.Prepare(ctx, opts)
	if err != nil {
		return err
	}
	if res == selector.ResultAllDead && !probe.WhitelistOnly() {
		if c.Store != nil && c.Store.WLBuiltinConnectEnabled(ctx) {
			c.Log.Info("open-network pool dead, retry whitelist/builtin pool", "event", "H22-retry")
			if c.Activity != nil {
				c.Activity(ctx, "Подписки недоступны, проверка WL-пула…")
			}
			opts.WhitelistOnly = true
			best, res, err = c.Selector.Prepare(ctx, opts)
		} else {
			c.Log.Info("subscription pool dead, wl builtin rescue disabled", "event", "H22-skip")
		}
	}
	if err != nil {
		return err
	}
	switch res {
	case selector.ResultNoProfiles:
		return fmt.Errorf("no profiles; run bootstrap or --import-uri")
	case selector.ResultAllDead:
		return fmt.Errorf("all subscription servers failed probes (WL builtin rescue is off; enable wl_builtin_connect in settings or add working subs)")
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
