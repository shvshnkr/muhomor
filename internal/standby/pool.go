package standby

import (
	"context"
	"sort"
	"time"

	"github.com/muhomor/muhomor/internal/store"
)

func filterFresh(entries []Entry, maxAge time.Duration, excludeID int64) []Entry {
	now := time.Now()
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.ProfileID <= 0 || e.ProfileID == excludeID {
			continue
		}
		if e.DelayMs <= 0 {
			continue
		}
		if !e.VerifiedAt.IsZero() && now.Sub(e.VerifiedAt) > maxAge {
			continue
		}
		out = append(out, e)
	}
	return out
}

// IsHotFresh reports whether hot pool has HotSize fresh URL-verified entries.
func IsHotFresh(ctx context.Context, st *store.Store) bool {
	hot := filterFresh(LoadHot(ctx, st), HotMaxAge, 0)
	return len(hot) >= HotSize
}

// PickBestHot returns the best fresh hot entry as a profile.
func PickBestHot(ctx context.Context, st *store.Store, excludeID int64) (store.Profile, bool) {
	wlOnly := st != nil && st.SimpleModeWLOnly(ctx)
	hot := filterFresh(LoadHot(ctx, st), HotMaxAge, excludeID)
	for _, e := range hot {
		p, err := st.ProfileByID(ctx, e.ProfileID)
		if err != nil || !p.Enabled {
			continue
		}
		if !blEligible(ctx, st, p, e, wlOnly) {
			continue
		}
		return p, true
	}
	return store.Profile{}, false
}

// PickWarm returns the best fresh warm entry as a profile.
func PickWarm(ctx context.Context, st *store.Store, excludeID int64) (store.Profile, bool) {
	wlOnly := st != nil && st.SimpleModeWLOnly(ctx)
	warm := filterFresh(LoadWarm(ctx, st), WarmMaxAge, excludeID)
	for _, e := range warm {
		p, err := st.ProfileByID(ctx, e.ProfileID)
		if err != nil || !p.Enabled {
			continue
		}
		if !blEligible(ctx, st, p, e, wlOnly) {
			continue
		}
		return p, true
	}
	return store.Profile{}, false
}

// HotAlternates returns hot entries excluding activeID (for failover).
func HotAlternates(ctx context.Context, st *store.Store, activeID int64) []Entry {
	return filterFresh(LoadHot(ctx, st), HotMaxAge, activeID)
}

// PromoteAfterConnect moves the connected profile to the front of hot with fresh verification.
func PromoteAfterConnect(ctx context.Context, st *store.Store, profileID int64, proxyName string, delayMs int) {
	if st == nil || profileID <= 0 {
		return
	}
	now := time.Now().UTC()
	entry := Entry{ProfileID: profileID, ProxyName: proxyName, DelayMs: delayMs, VerifiedAt: now}
	hot := LoadHot(ctx, st)
	out := []Entry{entry}
	for _, e := range hot {
		if e.ProfileID == profileID {
			continue
		}
		out = append(out, e)
		if len(out) >= HotSize {
			break
		}
	}
	_ = SaveHot(ctx, st, out)
}

// ApplyDelayResults writes top delays into hot (HotSize) and warm (WarmSize).
func ApplyDelayResults(ctx context.Context, st *store.Store, ranked []Entry) {
	if st == nil || len(ranked) == 0 {
		return
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].DelayMs == ranked[j].DelayMs {
			return ranked[i].ProfileID < ranked[j].ProfileID
		}
		return ranked[i].DelayMs < ranked[j].DelayMs
	})
	now := time.Now().UTC()
	for i := range ranked {
		if ranked[i].VerifiedAt.IsZero() {
			ranked[i].VerifiedAt = now
		}
	}
	hotN := HotSize
	if len(ranked) < hotN {
		hotN = len(ranked)
	}
	_ = SaveHot(ctx, st, ranked[:hotN])
	warmStart := hotN
	warmEnd := warmStart + WarmSize
	if warmEnd > len(ranked) {
		warmEnd = len(ranked)
	}
	if warmStart < warmEnd {
		_ = SaveWarm(ctx, st, ranked[warmStart:warmEnd])
	} else {
		_ = SaveWarm(ctx, st, nil)
	}
	TouchLastRefresh(ctx, st)
}

// ShortFallbackQueue builds hot+warm profile IDs for post-connect failover (max ~11).
func ShortFallbackQueue(ctx context.Context, st *store.Store, activeID int64) []int64 {
	seen := map[int64]struct{}{}
	if activeID > 0 {
		seen[activeID] = struct{}{}
	}
	var ids []int64
	appendID := func(id int64) {
		if id <= 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, e := range filterFresh(LoadHot(ctx, st), HotMaxAge, 0) {
		appendID(e.ProfileID)
	}
	for _, e := range filterFresh(LoadWarm(ctx, st), WarmMaxAge, 0) {
		appendID(e.ProfileID)
	}
	return ids
}
