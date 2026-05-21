package subscription

import (
	"context"
	"fmt"
	"sync"

	"github.com/muhomor/muhomor/internal/store"
)

// maxLinesPerGroup caps imported URIs per subscription refresh (protects selector/DB).
const maxLinesPerGroup = 400

// RefreshDue refreshes all subscription groups (scheduler / maintenance).
func (u *Updater) RefreshDue(ctx context.Context, internetOK bool) error {
	if !internetOK {
		return fmt.Errorf("subscription refresh skipped: no internet")
	}
	groups, err := u.Store.ListGroups(ctx)
	if err != nil {
		return err
	}
	var lastErr error
	for _, g := range groups {
		if g.Kind != store.GroupKindSubscription || g.SubscriptionLink == "" {
			continue
		}
		if _, err := u.RefreshGroup(ctx, g.ID); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// RefreshDueWL refreshes only White Bolt WL groups (BS / whitelist-only network).
func (u *Updater) RefreshDueWL(ctx context.Context, internetOK bool) error {
	if !internetOK {
		return fmt.Errorf("subscription refresh skipped: no internet")
	}
	ids, err := ListWhiteBoltWLGroupIDs(ctx, u.Store)
	if err != nil {
		return err
	}
	return u.refreshGroupsParallel(ctx, ids, 4)
}

// RefreshDueOpen refreshes Black/open subscription groups (normal network, not BS).
func (u *Updater) RefreshDueOpen(ctx context.Context, internetOK bool) error {
	if !internetOK {
		return fmt.Errorf("subscription refresh skipped: no internet")
	}
	ids, err := ListOpenNetworkSubscriptionGroupIDs(ctx, u.Store)
	if err != nil {
		return err
	}
	return u.refreshGroupsParallel(ctx, ids, 3)
}

func (u *Updater) refreshGroupsParallel(ctx context.Context, ids []int64, workers int) error {
	if len(ids) == 0 {
		return nil
	}
	if workers < 1 {
		workers = 1
	}
	if workers > len(ids) {
		workers = len(ids)
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var lastErr error
	for _, id := range ids {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(gid int64) {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := u.RefreshGroup(ctx, gid); err != nil {
				mu.Lock()
				lastErr = err
				mu.Unlock()
			}
		}(id)
	}
	wg.Wait()
	return lastErr
}
