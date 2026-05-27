package standby

import (
	"context"

	"github.com/muhomor/muhomor/internal/probe"
	"github.com/muhomor/muhomor/internal/store"
)

// TopCandidates returns up to n warm-alive profiles suitable for picker batch URL-test.
func TopCandidates(ctx context.Context, st *store.Store, n int, wlOnly bool) ([]store.Profile, error) {
	if st == nil || n <= 0 {
		return nil, nil
	}
	if n > CandidateBatchCap {
		n = CandidateBatchCap
	}
	cfg := probe.ConfigFromStore(ctx, st)
	maxAge := cfg.WarmMaxAge
	if maxAge <= 0 {
		maxAge = WarmMaxAge
	}
	warm, err := st.ListWarmAliveProfiles(ctx, maxAge, n*2)
	if err != nil {
		return nil, err
	}
	if len(warm) == 0 {
		return nil, nil
	}
	all, err := st.ListAllProfiles(ctx)
	if err != nil {
		return warm[:min(len(warm), n)], nil
	}
	pool := filterConnectPool(all, wlOnly)
	warm = intersectPool(warm, pool)
	if len(warm) > n {
		warm = warm[:n]
	}
	warm = filterCemetery(ctx, st, warm)
	return warm, nil
}

func filterConnectPool(all []store.Profile, wlOnly bool) []store.Profile {
	if !wlOnly {
		var subs []store.Profile
		for _, p := range all {
			if p.WLBuiltinPool {
				continue
			}
			if p.IsSubscriptionWhitelistMarked() {
				continue
			}
			subs = append(subs, p)
		}
		return subs
	}
	var head []store.Profile
	for _, p := range all {
		if p.WLBuiltinPool || p.IsSubscriptionWhitelistMarked() {
			head = append(head, p)
		}
	}
	return head
}

func intersectPool(warm, pool []store.Profile) []store.Profile {
	if len(pool) == 0 {
		return warm
	}
	allow := make(map[int64]struct{}, len(pool))
	for _, p := range pool {
		allow[p.ID] = struct{}{}
	}
	out := make([]store.Profile, 0, len(warm))
	for _, p := range warm {
		if _, ok := allow[p.ID]; ok {
			out = append(out, p)
		}
	}
	return out
}

func filterCemetery(ctx context.Context, st *store.Store, profiles []store.Profile) []store.Profile {
	out := make([]store.Profile, 0, len(profiles))
	for _, p := range profiles {
		meta, err := st.ProbeMetaByID(ctx, p.ID)
		if err != nil {
			out = append(out, p)
			continue
		}
		if meta.State == store.ProbeCemetery {
			continue
		}
		out = append(out, p)
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
