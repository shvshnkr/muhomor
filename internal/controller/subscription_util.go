package controller

import (
	"context"

	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

func newSubscriptionUpdater(st *store.Store) *subscription.Updater {
	return &subscription.Updater{
		Store: st,
		UserAgent: func(ctx context.Context, groupID int64) string {
			v, _ := st.GetKV(ctx, store.KeyGroupUserAgent(groupID))
			return v
		},
	}
}

func (r *Runtime) RefreshSubscriptionGroup(ctx context.Context, groupID int64) (int, error) {
	u := newSubscriptionUpdater(r.Store)
	return u.RefreshGroup(ctx, groupID)
}
