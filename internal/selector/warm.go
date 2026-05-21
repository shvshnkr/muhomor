package selector

import (
	"context"
	"fmt"
	"time"

	"github.com/muhomor/muhomor/internal/probe"
	"github.com/muhomor/muhomor/internal/store"
)

// applyBuiltinFallbackCap limits built-in share in fallback queue.
func applyBuiltinFallbackCap(ranked []store.Profile, maxPct int) []store.Profile {
	if maxPct <= 0 || len(ranked) == 0 {
		return ranked
	}
	var builtin, other []store.Profile
	for _, p := range ranked {
		if p.WLBuiltinPool {
			builtin = append(builtin, p)
		} else {
			other = append(other, p)
		}
	}
	if len(builtin) == 0 {
		return ranked
	}
	maxBuiltin := len(ranked) * maxPct / 100
	if maxBuiltin < 1 && maxPct > 0 && len(builtin) > 0 {
		maxBuiltin = 1
	}
	if maxBuiltin >= len(builtin) {
		return ranked
	}
	out := make([]store.Profile, 0, len(ranked))
	bUsed := 0
	for _, p := range ranked {
		if p.WLBuiltinPool {
			if bUsed >= maxBuiltin {
				continue
			}
			bUsed++
		}
		out = append(out, p)
	}
	return out
}

func (s *Selector) warmPrepare(ctx context.Context, pool []store.Profile, opts PrepareOpts, priority map[int64]struct{}) (store.Profile, PrepareResult, bool) {
	if s.Store == nil || !s.Store.ProbeWarmSelectEnabled(ctx) {
		return store.Profile{}, ResultNoProfiles, false
	}
	cfg := probe.ConfigFromStore(ctx, s.Store)
	warm, err := s.Store.ListWarmAliveProfiles(ctx, cfg.WarmMaxAge, 64)
	if err != nil || len(warm) == 0 {
		return store.Profile{}, ResultNoProfiles, false
	}
	before := len(warm)
	if s.Activity != nil {
		s.Activity(ctx, fmt.Sprintf("Warm: TCP-проверка %d…", min(len(warm), cfg.WarmSpotCap)))
	}
	warm = s.warmSpotFilter(ctx, warm, cfg.WarmSpotCap)
	if len(warm) == 0 {
		_ = s.Store.SetLastSelectReason(ctx, "warm:all_stale_after_spot")
		return store.Profile{}, ResultNoProfiles, false
	}
	if s.Log != nil {
		s.Log.Info("warm select", "warm", len(warm), "before_spot", before, "pool", len(pool), "event", "2K-warm")
	}
	tcpPings := map[int64]int{}
	for _, p := range warm {
		if p.Ping > 0 {
			tcpPings[p.ID] = p.Ping
		} else if p.LastDelayMs > 0 {
			tcpPings[p.ID] = p.LastDelayMs
		}
	}
	urlDelays := map[int64]int{}
	if s.Ephemeral != nil && len(warm) > 0 {
		cap := cfg.WarmURLCap
		if cap <= 0 {
			cap = 16
		}
		if opts.NetworkHandoff && cap > 12 {
			cap = 12
		}
		sub := warm
		if len(sub) > cap {
			sub = sub[:cap]
		}
		if s.Activity != nil {
			s.Activity(ctx, "Проверка warm-серверов (URL)…")
		}
		for _, p := range sub {
			ms, err := s.Ephemeral.TestProfile(ctx, p)
			if err == nil && ms > 0 {
				urlDelays[p.ID] = ms
			}
		}
	}
	ranked := rankProfiles(warm, tcpPings, urlDelays, priority, s.isCooldown)
	ranked = s.applyMultipathRank(ctx, ranked, tcpPings, urlDelays, priority, false)
	if len(ranked) == 0 {
		_ = s.Store.SetLastSelectReason(ctx, "warm:rank_empty")
		return store.Profile{}, ResultAllDead, false
	}
	maxPct := s.Store.EffectiveBuiltinFallbackMaxPct(ctx)
	ranked = applyBuiltinFallbackCap(ranked, maxPct)
	ids := make([]int64, len(ranked))
	for i, p := range ranked {
		ids[i] = p.ID
	}
	_ = s.Store.SetFallbackQueue(ctx, ids)
	best := ranked[0]
	_ = s.Store.SetSelectedProxy(ctx, best.ID)
	_ = s.Store.SetLastKnownGood(ctx, best.ID)
	reason := fmt.Sprintf("warm:best=%d queue=%d spot=%d url_ok=%d", best.ID, len(ranked), min(before, cfg.WarmSpotCap), len(urlDelays))
	_ = s.Store.SetLastSelectReason(ctx, reason)
	if s.Log != nil {
		s.Log.Info("queue prepared", "best", best.ID, "size", len(ranked), "warm", true, "event", "H4")
	}
	return best, ResultSuccess, true
}

// warmSpotFilter re-validates top warm rows with live TCP (avoids stale persisted alive).
func (s *Selector) warmSpotFilter(ctx context.Context, warm []store.Profile, spotCap int) []store.Profile {
	if spotCap <= 0 || len(warm) == 0 {
		return warm
	}
	toCheck := warm
	if len(toCheck) > spotCap {
		toCheck = toCheck[:spotCap]
	}
	live := s.tcpProbeAll(ctx, toCheck, nil, false)
	checked := map[int64]struct{}{}
	for _, p := range toCheck {
		checked[p.ID] = struct{}{}
	}
	out := make([]store.Profile, 0, len(warm))
	for _, p := range warm {
		if _, ok := checked[p.ID]; !ok {
			out = append(out, p)
			continue
		}
		if ms, liveOK := live[p.ID]; liveOK && ms > 0 {
			out = append(out, p)
			continue
		}
		s.warmMarkStale(ctx, p.ID)
	}
	return out
}

func (s *Selector) warmMarkStale(ctx context.Context, id int64) {
	if s.Store == nil {
		return
	}
	m, err := s.Store.ProbeMetaByID(ctx, id)
	if err != nil {
		return
	}
	now := time.Now()
	m.State = store.ProbeSuspect
	m.LastFailAt = now
	m.LastCheckedAt = now
	m.FailStreak++
	m.LastErrorClass = "warm_spot_stale"
	_ = s.Store.UpdateProfileProbeMeta(ctx, id, m)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
