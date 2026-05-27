package controller

import (
	"context"
	"time"

	"github.com/muhomor/muhomor/internal/store"
)

// pingSampleTTL balances live latency vs mihomo delay probe cost (5–10s timeout).
const pingSampleTTL = 5 * time.Second

// pingGraceAfterConnect avoids red ping while mihomo/geo settle after H30.
const pingGraceAfterConnect = 18 * time.Second

func (r *Runtime) startPingSampler(ctx context.Context) {
	r.pingMu.Lock()
	if r.pingStop != nil {
		r.pingMu.Unlock()
		return
	}
	stop := make(chan struct{})
	r.pingStop = stop
	r.pingMu.Unlock()

	go func() {
		r.clearLastPing(ctx)
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-time.After(pingGraceAfterConnect):
		}
		tick := time.NewTicker(pingSampleTTL)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-tick.C:
				if r.connectInFlight() {
					continue
				}
				r.samplePingOnce(ctx)
			}
		}
	}()
}

func (r *Runtime) samplePingOnce(ctx context.Context) {
	r.pingMu.Lock()
	if r.pingBusy {
		r.pingMu.Unlock()
		return
	}
	r.pingBusy = true
	r.pingMu.Unlock()
	defer func() {
		r.pingMu.Lock()
		r.pingBusy = false
		r.pingMu.Unlock()
	}()

	r.mu.Lock()
	client := r.mihomo
	proxy := r.proxy
	r.mu.Unlock()
	if client == nil || proxy == "" {
		return
	}

	pingCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if r.connectInFlight() {
		return
	}
	_, _ = r.Ping(pingCtx)
	r.publishStatusEvent("ping")
}

func (r *Runtime) stopPingSampler() {
	r.pingMu.Lock()
	stop := r.pingStop
	r.pingStop = nil
	r.pingMu.Unlock()
	if stop != nil {
		close(stop)
	}
}

func (r *Runtime) clearLastPing(ctx context.Context) {
	if r.Store == nil {
		return
	}
	_ = r.Store.SetKV(ctx, store.KeyLastServicePingMs, "")
	_ = r.Store.SetKV(ctx, store.KeyLastServicePingError, "")
}
