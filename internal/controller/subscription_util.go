package controller

import (
	"context"

	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

func newSubscriptionUpdater(st *store.Store) *subscription.Updater {
	return &subscription.Updater{Store: st}
}

func (r *Runtime) RefreshSubscriptionGroup(ctx context.Context, groupID int64) (int, error) {
	return newSubscriptionUpdater(r.Store).RefreshGroup(ctx, groupID)
}
