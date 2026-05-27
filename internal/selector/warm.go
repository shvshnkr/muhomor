package selector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/muhomor/muhomor/internal/probe"
	"github.com/muhomor/muhomor/internal/probe/blexit"
	"github.com/muhomor/muhomor/internal/standby"
	"github.com/muhomor/muhomor/internal/store"
)

// applyBuiltinFallbackCap limits built-in share; excess builtins move to tail (not dropped).
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
	if maxBuiltin < 1 && maxPct > 0 {
		maxBuiltin = 1
	}
	if maxBuiltin > len(builtin) {
		maxBuiltin = len(builtin)
	}
	return append(append([]store.Profile{}, other...), builtin[:maxBuiltin]...)
}

func (s *Selector) warmPrepare(ctx context.Context, pool []store.Profile, opts PrepareOpts, priority map[int64]struct{}) (store.Profile, PrepareResult, bool) {
	if s.Store == nil || !s.Store.ProbeWarmSelectEnabled(ctx) {
		return store.Profile{}, ResultNoProfiles, false
	}
	if best, res, ok := s.warmPrepareHotDelegate(ctx, pool); ok {
		return best, res, true
	}
	cfg := probe.ConfigFromStore(ctx, s.Store)
	warm, err := s.Store.ListWarmAliveProfiles(ctx, cfg.WarmMaxAge, 64)
	if err != nil || len(warm) == 0 {
		return store.Profile{}, ResultNoProfiles, false
	}
	warm = filterWarmByPool(warm, pool)
	if len(warm) == 0 {
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
	urlAttempted := 0
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
		urlAttempted = len(sub)
		urlDelays = s.warmURLTest(ctx, sub, cap)
	}
	s.recordTelegramBatch(urlAttempted, len(urlDelays))
	targetCircuitOpen := s.telegramCircuitOpen()
	blPass, blEnabled := blexit.PassResult{}, false
	if len(urlDelays) > 0 {
		urlDelays, blPass, blEnabled = s.applyBLExitFilter(ctx, warm, urlDelays, opts.WhitelistOnly)
	}
	if blEnabled && blPass.Scope == blexit.ScopeAll && len(urlDelays) == 0 {
		_ = s.Store.SetLastSelectReason(ctx, "warm:bl_all_failed"+blexit.ReasonSuffix(blPass, true))
		return store.Profile{}, ResultAllDead, false
	}
	blOK := blOkMap(blPass, blEnabled)
	blUplinkOnly := blUplinkOnlyScope(blPass, blEnabled)
	decision := classifyPrepareDecision(len(tcpPings), len(urlDelays))
	degraded := decision == PrepareDecisionDegraded
	s.setProbeEvidence(ProbeEvidence{
		TCPOK:     len(tcpPings),
		URLOK:     len(urlDelays),
		PoolSize:  len(warm),
		Degraded:  degraded,
		PrepareDecision: string(decision),
		TelegramTargetCircuit: circuitLabel(targetCircuitOpen),
		UpdatedAt: time.Now(),
	})
	ranked := rankProfiles(warm, tcpPings, urlDelays, blOK, blUplinkOnly, priority, s.isCooldown, degraded && !targetCircuitOpen, nil)
	ranked = s.applyMultipathRank(ctx, ranked, tcpPings, urlDelays, priority, false)
	persistURLAlivePool(ctx, s.Store, urlDelays)
	if len(ranked) == 0 {
		_ = s.Store.SetLastSelectReason(ctx, "warm:rank_empty")
		return store.Profile{}, ResultAllDead, false
	}
	maxPct := s.Store.EffectiveBuiltinFallbackMaxPct(ctx)
	ranked = applyBuiltinFallbackCap(ranked, maxPct)
	ranked = capRankedForFallback(ctx, s.Store, ranked, tcpPings, urlDelays, blEnabled && blPass.FailClosed)
	if len(ranked) == 0 {
		_ = s.Store.SetLastSelectReason(ctx, "warm:cap_empty")
		return store.Profile{}, ResultAllDead, false
	}
	ids := make([]int64, len(ranked))
	for i, p := range ranked {
		ids[i] = p.ID
	}
	_ = s.Store.SetFallbackQueue(ctx, ids)
	best := ranked[0]
	_ = s.Store.SetSelectedProxy(ctx, best.ID)
	_ = s.Store.SetLastKnownGood(ctx, best.ID)
	reason := fmt.Sprintf("warm:best=%d queue=%d spot=%d prepare_decision=%s tcp_ok=%d url_ok=%d telegram_target_circuit=%s degraded=%t%s", best.ID, len(ranked), min(before, cfg.WarmSpotCap), decision, len(tcpPings), len(urlDelays), circuitLabel(targetCircuitOpen), degraded, blexit.ReasonSuffix(blPass, blEnabled))
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
	live := s.tcpProbeAll(ctx, toCheck, nil, false, -1, OwnerConnect)
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
	m.LastFailAt = now
	m.LastCheckedAt = now
	m.FailStreak++
	m.LastErrorClass = "warm_spot_stale"
	if m.FailStreak >= 6 {
		m.State = store.ProbeCemetery
	} else if m.FailStreak >= 2 {
		m.State = store.ProbeDead
	} else {
		m.State = store.ProbeSuspect
	}
	m.NextProbeAt = probe.NextProbeAfter(m.State, m.FailStreak, now)
	_ = s.Store.UpdateProfileProbeMeta(ctx, id, m)
}

// warmPrepareHotDelegate skips spot+URL when standby hot pool is fresh.
func (s *Selector) warmPrepareHotDelegate(ctx context.Context, pool []store.Profile) (store.Profile, PrepareResult, bool) {
	if s.Store == nil || !standby.IsHotFresh(ctx, s.Store) {
		return store.Profile{}, ResultNoProfiles, false
	}
	hot, ok := standby.PickBestHot(ctx, s.Store, 0)
	if !ok {
		return store.Profile{}, ResultNoProfiles, false
	}
	if len(pool) > 0 && !profileInPool(hot.ID, pool) {
		return store.Profile{}, ResultNoProfiles, false
	}
	q := standby.ShortFallbackQueue(ctx, s.Store, hot.ID)
	if len(q) == 0 {
		q = []int64{hot.ID}
	}
	_ = s.Store.SetFallbackQueue(ctx, q)
	_ = s.Store.SetSelectedProxy(ctx, hot.ID)
	_ = s.Store.SetLastKnownGood(ctx, hot.ID)
	reason := fmt.Sprintf("warm:hot_delegate=%d queue=%d", hot.ID, len(q))
	_ = s.Store.SetLastSelectReason(ctx, reason)
	s.setProbeEvidence(ProbeEvidence{
		TCPOK:     1,
		URLOK:     1,
		PoolSize:  len(pool),
		Degraded:  false,
		UpdatedAt: time.Now(),
	})
	if s.Log != nil {
		s.Log.Info("queue prepared", "best", hot.ID, "size", len(q), "hot_delegate", true, "event", "H4")
	}
	if s.Activity != nil {
		s.Activity(ctx, "Warm: hot standby готов")
	}
	return hot, ResultSuccess, true
}

func (s *Selector) warmURLTest(ctx context.Context, profiles []store.Profile, cap int) map[int64]int {
	if s.Ephemeral == nil || len(profiles) == 0 {
		return nil
	}
	sub := profiles
	if len(sub) > cap {
		sub = sub[:cap]
	}
	if s.Activity != nil {
		s.Activity(ctx, fmt.Sprintf("Проверка warm-серверов (URL) %d…", len(sub)))
	}
	usePicker := pickerReady(ctx, s.Ephemeral.Picker)
	if usePicker {
		if out, ok := s.Ephemeral.testProfilesPickerPrimary(ctx, sub); ok {
			return out
		}
	}
	var mu sync.Mutex
	sem := make(chan struct{}, urlTestWorkers)
	var wg sync.WaitGroup
	out := map[int64]int{}
	for _, p := range sub {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			ms, err := s.Ephemeral.TestProfile(ctx, p)
			if err != nil || ms <= 0 {
				return
			}
			mu.Lock()
			out[p.ID] = ms
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

func profileInPool(id int64, pool []store.Profile) bool {
	for _, p := range pool {
		if p.ID == id {
			return true
		}
	}
	return false
}

func filterWarmByPool(warm, pool []store.Profile) []store.Profile {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
