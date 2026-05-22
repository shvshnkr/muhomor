package simplemode

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/muhomor/muhomor/internal/reachability"
	"github.com/muhomor/muhomor/internal/selector"
	"github.com/muhomor/muhomor/internal/store"
)

const adaptDebounceMs = 2500

// Adaptor handles network handoff (SimpleModeVpnCoordinator parity).
type Adaptor struct {
	Store     *store.Store
	Selector  *selector.Selector
	Probe     func(context.Context, bool) reachability.Result
	Reselect  func(context.Context, store.Profile, bool) error
	Log       *slog.Logger
	WLRescue  func(context.Context) bool // optional: wl_builtin_connect enabled

	mu          sync.Mutex
	lastAdaptAt int64
	adaptGen    atomic.Int32
	jobCancel   context.CancelFunc
}

// CancelAll stops in-flight adaptation (daemon shutdown).
func (a *Adaptor) CancelAll() {
	if a == nil || a.Selector == nil {
		return
	}
	a.Selector.CancelAdapt()
	a.adaptGen.Add(1)
	if a.jobCancel != nil {
		a.jobCancel()
	}
}

func (a *Adaptor) ScheduleAdaptation(ctx context.Context, reason string) {
	if a == nil || a.Selector == nil {
		return
	}
	a.Selector.CancelAdapt()
	gen := a.adaptGen.Add(1)
	if a.jobCancel != nil {
		a.jobCancel()
	}
	runCtx, cancel := context.WithCancel(ctx)
	a.jobCancel = cancel
	go func() {
		defer cancel()
		a.mu.Lock()
		now := time.Now().UnixMilli()
		wait := adaptDebounceMs - (now - a.lastAdaptAt)
		a.mu.Unlock()
		if wait > 0 {
			select {
			case <-time.After(time.Duration(wait) * time.Millisecond):
			case <-runCtx.Done():
				return
			}
		}
		if gen != a.adaptGen.Load() {
			return
		}
		a.mu.Lock()
		a.lastAdaptAt = time.Now().UnixMilli()
		a.mu.Unlock()
		if err := a.adaptLocked(runCtx, reason, gen); err != nil && a.Log != nil {
			a.Log.Warn("adapt failed", "reason", reason, "err", err, "event", "H30")
		}
	}()
}

func (a *Adaptor) adaptLocked(ctx context.Context, reason string, gen int32) error {
	if gen != a.adaptGen.Load() {
		return context.Canceled
	}
	probe := a.Probe(ctx, true)
	_ = a.Store.SetKV(ctx, store.KeyActiveWhitelistRestricted, boolStr(probe.WhitelistOnly()))
	_ = a.Store.SetKV(ctx, store.KeySimpleModeUseWLPoolOnly, boolStr(probe.WhitelistOnly()))
	if !probe.AnyReachable() {
		if a.Log != nil {
			a.Log.Info("adapt skipped unreachable", "reason", reason, "event", "H30")
		}
		return nil
	}
	opts := selector.PrepareOpts{
		Owner:          selector.OwnerAdapt,
		NetworkHandoff: true,
		WhitelistOnly:  probe.WhitelistOnly(),
	}
	best, res, err := a.Selector.Prepare(ctx, opts)
	if err != nil {
		return err
	}
	if res == selector.ResultAllDead && !probe.WhitelistOnly() && a.wlRescueEnabled(ctx) {
		if a.Log != nil {
			a.Log.Info("adapt open pool dead, retry wl pool", "event", "H22-retry")
		}
		opts.WhitelistOnly = true
		best, res, err = a.Selector.Prepare(ctx, opts)
	}
	if err != nil {
		return err
	}
	if res != selector.ResultSuccess {
		return nil
	}
	if gen != a.adaptGen.Load() {
		return context.Canceled
	}
	return a.Reselect(ctx, best, probe.WhitelistOnly())
}

func (a *Adaptor) wlRescueEnabled(ctx context.Context) bool {
	if a.WLRescue != nil {
		return a.WLRescue(ctx)
	}
	return a.Store != nil && a.Store.WLBuiltinConnectEnabled(ctx)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
