package controller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/muhomor/muhomor/internal/reachability"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

const (
	assetUpdateInterval        = 24 * time.Hour
	subscriptionRefreshInterval = 45 * time.Minute
)

// Scheduler runs periodic background tasks (route assets, subscription refresh).
type Scheduler struct {
	Store   *store.Store
	Assets  *subscription.AssetUpdater
	Subs    *subscription.Updater
	Log     *slog.Logger
}

func (s *Scheduler) Run(ctx context.Context) {
	tick := time.NewTicker(1 * time.Hour)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	s.tickAssets(ctx)
	s.tickSubscriptions(ctx)
}

func (s *Scheduler) tickAssets(ctx context.Context) {
	lastStr, _ := s.Store.GetKV(ctx, store.KeyLastAssetUpdateAt)
	if lastStr != "" {
		if t, err := time.Parse(time.RFC3339, lastStr); err == nil {
			if time.Since(t) < assetUpdateInterval {
				return
			}
		}
	}
	if s.Assets != nil {
		if err := s.Assets.UpdateIfDue(ctx); err != nil {
			if s.Log != nil {
				s.Log.Warn("asset update", "err", err)
			}
			return
		}
	}
	_ = s.Store.SetKV(ctx, store.KeyLastAssetUpdateAt, time.Now().Format(time.RFC3339))
	if s.Log != nil {
		s.Log.Info("asset update tick", "event", "route-asset")
	}
}

func (s *Scheduler) tickSubscriptions(ctx context.Context) {
	lastStr, _ := s.Store.GetKV(ctx, store.KeyLastBackgroundSubRefreshAt)
	if lastStr != "" {
		var lastMs int64
		if _, err := fmt.Sscan(lastStr, &lastMs); err == nil && lastMs > 0 {
			if time.Since(time.UnixMilli(lastMs)) < subscriptionRefreshInterval {
				return
			}
		}
	}
	if s.Subs == nil {
		return
	}
	probe := reachability.Probe(ctx, true)
	if !probe.AnyReachable() {
		return
	}
	var err error
	if probe.WhitelistOnly() {
		err = s.Subs.RefreshDueWL(ctx, true)
	} else {
		err = s.Subs.RefreshDueOpen(ctx, true)
	}
	if err != nil && s.Log != nil {
		s.Log.Warn("scheduled sub refresh", "err", err, "wl_only", probe.WhitelistOnly())
	}
	_ = s.Store.SetKV(ctx, store.KeyLastBackgroundSubRefreshAt, fmt.Sprintf("%d", time.Now().UnixMilli()))
	if s.Log != nil {
		s.Log.Info("subscription refresh tick", "event", "H29-scheduler")
	}
}
