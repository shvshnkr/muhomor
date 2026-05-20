package controller

import (
	"context"
	"log/slog"
	"time"

	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

const assetUpdateInterval = 24 * time.Hour

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
	lastStr, _ := s.Store.GetKV(ctx, store.KeyLastAssetUpdateAt)
	if lastStr != "" {
		if t, err := time.Parse(time.RFC3339, lastStr); err == nil {
			if time.Since(t) < assetUpdateInterval {
				return
			}
		}
	}
	if s.Assets != nil {
		if err := s.Assets.UpdateIfDue(ctx); err != nil && s.Log != nil {
			s.Log.Warn("asset update", "err", err)
		}
	}
	_ = s.Store.SetKV(ctx, store.KeyLastAssetUpdateAt, time.Now().Format(time.RFC3339))
	if s.Log != nil {
		s.Log.Info("asset update tick", "event", "route-asset")
	}
}
