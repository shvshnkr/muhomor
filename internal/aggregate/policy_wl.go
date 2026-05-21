package aggregate

import "github.com/muhomor/muhomor/internal/store"

// WLPolicy controls built-in WL pool in multipath channel pool.
type WLPolicy struct {
	EmergencyOnly bool
	MaxPct        int
	SubsDegraded  bool
}

// FilterChannelPool applies WL emergency-only and builtin cap.
func FilterChannelPool(ranked []store.Profile, pol WLPolicy) []store.Profile {
	if len(ranked) == 0 {
		return ranked
	}
	if !pol.EmergencyOnly {
		return capBuiltinShare(ranked, pol.MaxPct)
	}
	if !pol.SubsDegraded {
		var out []store.Profile
		for _, p := range ranked {
			if !p.WLBuiltinPool {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return capBuiltinShare(ranked, pol.MaxPct)
}

func capBuiltinShare(ranked []store.Profile, maxPct int) []store.Profile {
	if maxPct <= 0 || maxPct >= 100 {
		return ranked
	}
	maxB := len(ranked) * maxPct / 100
	if maxB < 1 {
		maxB = 1
	}
	bUsed := 0
	out := make([]store.Profile, 0, len(ranked))
	for _, p := range ranked {
		if p.WLBuiltinPool {
			if bUsed >= maxB {
				continue
			}
			bUsed++
		}
		out = append(out, p)
	}
	return out
}

// SubsPoolDegraded true when no non-WL candidates look healthy.
func SubsPoolDegraded(scores []ChannelScore, profiles []store.Profile) bool {
	healthy := 0
	wl := map[int64]bool{}
	for _, p := range profiles {
		wl[p.ID] = p.WLBuiltinPool
	}
	for _, sc := range scores {
		if wl[sc.ProfileID] {
			continue
		}
		if sc.Score < 1<<28 {
			healthy++
		}
	}
	return healthy < 2
}
