package selector

import (
	"context"
	"sort"

	"github.com/muhomor/muhomor/internal/aggregate"
	"github.com/muhomor/muhomor/internal/store"
)

// rankMultipath reorders pool using channel score (goodput/latency/stability).
func (s *Selector) rankMultipath(ctx context.Context, pool []store.Profile, tcp, url map[int64]int, priority map[int64]struct{}, wlOnly bool) ([]store.Profile, aggregate.ScheduleResult, bool) {
	if s.Store == nil || !s.Store.MultipathEnabled(ctx) {
		return nil, aggregate.ScheduleResult{}, false
	}
	preset := s.Store.MultipathPreset(ctx)
	cfg := aggregate.ConfigForPreset(preset)
	sch := &aggregate.Scheduler{Config: cfg}
	scoringPool := pool
	if len(url) > 0 {
		urlLive := make([]store.Profile, 0, len(url))
		for _, p := range pool {
			if url[p.ID] > 0 {
				urlLive = append(urlLive, p)
			}
		}
		// Flow aggregation should not seed PROXY_BULK with TCP-only survivors when
		// URL batch produced live candidates. Keep the rest in fallback order below.
		if len(urlLive) >= 2 {
			scoringPool = urlLive
		}
	}
	metrics := func(id int64) store.ChannelMetrics {
		m, _ := s.Store.ChannelMetricsByID(ctx, id)
		return m
	}
	inputs := aggregate.BuildInputs(scoringPool, tcp, url, priority, s.isCooldown, metrics)
	flow := aggregate.ClassifyFlow(aggregate.ClassifyOpts{BulkHint: len(pool) > 12})
	wlBuiltin := s.Store != nil && s.Store.WLBuiltinConnectEnabled(ctx)
	pol := aggregate.WLPolicy{
		EmergencyOnly: (s.Store.MultipathWLEmergencyOnly(ctx) || !wlBuiltin) && !wlOnly,
		MaxPct:        s.Store.EffectiveBuiltinFallbackMaxPct(ctx),
	}
	res := sch.Schedule(scoringPool, inputs, flow, pol)
	if res.PrimaryID == 0 && len(res.ChannelPool) == 0 {
		return nil, res, false
	}
	byID := map[int64]store.Profile{}
	for _, p := range pool {
		byID[p.ID] = p
	}
	var ordered []store.Profile
	for _, id := range res.ChannelPool {
		if p, ok := byID[id]; ok {
			ordered = append(ordered, p)
		}
	}
	// append any remaining by score order
	sort.SliceStable(res.Scores, func(i, j int) bool { return res.Scores[i].Score < res.Scores[j].Score })
	seen := map[int64]struct{}{}
	for _, p := range ordered {
		seen[p.ID] = struct{}{}
	}
	for _, sc := range res.Scores {
		if _, ok := seen[sc.ProfileID]; ok {
			continue
		}
		if p, ok := byID[sc.ProfileID]; ok {
			ordered = append(ordered, p)
			seen[p.ID] = struct{}{}
		}
	}
	for _, p := range pool {
		if _, ok := seen[p.ID]; ok {
			continue
		}
		ordered = append(ordered, p)
		seen[p.ID] = struct{}{}
	}
	_ = s.Store.SetMultipathChannelPool(ctx, res.ChannelPool)
	_ = s.Store.SetMultipathLastReason(ctx, res.Reason)
	healthy := aggregate.CountHealthy(res.Scores)
	_ = s.Store.SetMultipathStats(ctx, store.MultipathStats{
		Enabled:         true,
		Preset:          preset,
		WLEmergencyOnly: pol.EmergencyOnly,
		ActiveChannels:  len(res.ChannelPool),
		HealthyChannels: healthy,
		LastReason:      res.Reason,
	})
	if s.Log != nil {
		s.Log.Info("multipath schedule", "primary", res.PrimaryID, "pool", len(res.ChannelPool), "reason", res.Reason, "event", "MP-schedule")
	}
	return ordered, res, true
}

func (s *Selector) applyMultipathRank(ctx context.Context, ranked []store.Profile, tcp, url map[int64]int, priority map[int64]struct{}, wlOnly bool) []store.Profile {
	if mp, res, ok := s.rankMultipath(ctx, ranked, tcp, url, priority, wlOnly); ok && len(mp) > 0 {
		if res.PrimaryID > 0 {
			// move primary to front
			for i, p := range mp {
				if p.ID == res.PrimaryID {
					if i > 0 {
						out := append([]store.Profile{mp[i]}, append(mp[:i], mp[i+1:]...)...)
						return out
					}
					break
				}
			}
		}
		return mp
	}
	return ranked
}
