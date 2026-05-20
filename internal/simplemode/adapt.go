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

	mu          sync.Mutex
	lastAdaptAt int64
	adaptGen    atomic.Int32
	jobCancel   context.CancelFunc
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
		if now-a.lastAdaptAt < adaptDebounceMs {
			a.mu.Unlock()
			a.Log.Info("adapt skipped debounce", "reason", reason, "event", "H30")
			return
		}
		a.lastAdaptAt = now
		a.mu.Unlock()
		if err := a.adaptLocked(runCtx, reason, gen); err != nil {
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
	if !probe.AnyReachable() {
		return nil
	}
	best, res, err := a.Selector.Prepare(ctx, selector.PrepareOpts{
		Owner:          selector.OwnerAdapt,
		NetworkHandoff: true,
		WhitelistOnly:  probe.WhitelistOnly(),
	})
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

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
