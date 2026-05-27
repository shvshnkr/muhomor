package selector

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/muhomor/muhomor/internal/activity"
	"github.com/muhomor/muhomor/internal/bootstrap"
	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/profileclass"
	"github.com/muhomor/muhomor/internal/probe/blexit"
	"github.com/muhomor/muhomor/internal/store"
)

const (
	openNetTCPProbeCap       = 128
	profileFailureCooldownMs = 30 * 60 * 1000
	urlTestCapHandoff        = 12
	urlTestCapDefault        = 70 // cold path without picker; with picker capped at urlTestCapPicker
	tcpProbeWorkers          = 48
	urlTestWorkers           = 24
)

// Owner mirrors PrepareOwner in Kotlin.
type Owner int

const (
	OwnerConnect Owner = iota
	OwnerAdapt
	OwnerSessionRecover
)

// DelayTester measures proxy latency (mihomo /proxies/{name}/delay).
type DelayTester interface {
	TestProxyDelay(ctx context.Context, proxyName string) (int, error)
}

// Selector implements AutoServerSelector parity (Phase 2 subset).
type Selector struct {
	Store     *store.Store
	Log       *slog.Logger
	TCPTimeout time.Duration
	Ephemeral *EphemeralTester
	Activity  activity.Sink

	connectGen atomic.Int32
	adaptGen   atomic.Int32
	failures   sync.Map // id -> failedAt ms
	probeMu    sync.Mutex
	probeEv    ProbeEvidence
	targetMu   sync.Mutex
	targetWinStart time.Time
	targetURLAttempts int
	targetURLSuccesses int
	targetCircuitOpenUntil time.Time
}

type ProbeEvidence struct {
	TCPOK     int
	URLOK     int
	PoolSize  int
	Degraded  bool
	PrepareDecision string
	TelegramTargetCircuit string
	UpdatedAt time.Time
}

type PrepareDecision string

const (
	PrepareDecisionHardDead PrepareDecision = "hard_dead"
	PrepareDecisionDegraded PrepareDecision = "degraded"
	PrepareDecisionOK       PrepareDecision = "ok"
)

type PrepareOpts struct {
	Owner          Owner
	NetworkHandoff bool
	WhitelistOnly  bool
	Tester         DelayTester
	ProxyNames     map[int64]string // profile id -> mihomo proxy name
}

type PrepareResult int

const (
	ResultSuccess PrepareResult = iota
	ResultNoProfiles
	ResultAllDead
)

func (s *Selector) CancelConnect() { s.connectGen.Add(1) }
func (s *Selector) CancelAdapt()   { s.adaptGen.Add(1) }

func (s *Selector) Prepare(ctx context.Context, opts PrepareOpts) (store.Profile, PrepareResult, error) {
	gen := s.newGen(opts.Owner)
	if s.needWLBuiltin(ctx, opts.WhitelistOnly) {
		_, _ = bootstrap.EnsureWLBuiltin(ctx, s.Store)
	}
	profiles, err := s.Store.ListAllProfiles(ctx)
	if err != nil {
		return store.Profile{}, ResultNoProfiles, err
	}
	profiles = filterEnabled(profiles)
	if len(profiles) == 0 {
		return store.Profile{}, ResultNoProfiles, nil
	}

	selectedBefore, _ := s.Store.SelectedProxy(ctx)
	handoff := opts.NetworkHandoff || opts.Owner == OwnerSessionRecover
	priority := s.handoffPriority(ctx, selectedBefore, handoff)
	pool, priorityIDs := s.buildPool(ctx, profiles, opts.WhitelistOnly, priority)
	pool = s.filterCemeteryFromPool(ctx, pool, priorityIDs)

	if s.stale(gen, opts.Owner) {
		return store.Profile{}, ResultNoProfiles, context.Canceled
	}

	if s.Activity != nil {
		s.Activity(ctx, fmt.Sprintf("Подбор сервера: пул %d профилей…", len(pool)))
	}
	if best, res, ok := s.warmPrepare(ctx, pool, opts, priorityIDs); ok {
		return best, res, nil
	}
	if s.stale(gen, opts.Owner) {
		return store.Profile{}, ResultNoProfiles, context.Canceled
	}
	tcpPings, urlDelays := s.coldPrepareProbes(ctx, pool, opts, priorityIDs, gen)
	urlAttempted := len(urlTestCandidates(pool, tcpPings, priorityIDs, urlTestCapDefault))
	if opts.NetworkHandoff && urlAttempted > urlTestCapHandoff {
		urlAttempted = urlTestCapHandoff
	}
	s.recordTelegramBatch(urlAttempted, len(urlDelays))
	targetCircuitOpen := s.telegramCircuitOpen()

	if s.stale(gen, opts.Owner) {
		return store.Profile{}, ResultNoProfiles, context.Canceled
	}

	blPass, blEnabled := blexit.PassResult{}, false
	if len(urlDelays) > 0 {
		urlDelays, blPass, blEnabled = s.applyBLExitFilter(ctx, pool, urlDelays, opts.WhitelistOnly)
	}
	if blEnabled && blPass.Scope == blexit.ScopeAll && len(urlDelays) == 0 {
		if s.Log != nil {
			s.Log.Warn("BL filter removed all URL survivors; refusing connect", "tested", blPass.TestedCount, "scope", blPass.Scope, "event", "BL-exit-filter")
		}
		_ = s.Store.SetLastSelectReason(ctx, "live:bl_all_failed"+blexit.ReasonSuffix(blPass, true))
		return store.Profile{}, ResultAllDead, nil
	}

	decision := classifyPrepareDecision(len(tcpPings), len(urlDelays))
	if decision == PrepareDecisionHardDead && len(pool) > 0 {
		s.setProbeEvidence(ProbeEvidence{
			TCPOK: 0, URLOK: 0, PoolSize: len(pool), Degraded: false,
			PrepareDecision: string(decision), TelegramTargetCircuit: circuitLabel(targetCircuitOpen), UpdatedAt: time.Now(),
		})
		s.Log.Warn("all probes dead", "count", len(pool), "tcp", 0, "url", 0, "event", "H22")
		return store.Profile{}, ResultAllDead, nil
	}
	s.setProbeEvidence(ProbeEvidence{
		TCPOK:     len(tcpPings),
		URLOK:     len(urlDelays),
		PoolSize:  len(pool),
		Degraded:  decision == PrepareDecisionDegraded,
		PrepareDecision: string(decision),
		TelegramTargetCircuit: circuitLabel(targetCircuitOpen),
		UpdatedAt: time.Now(),
	})
	if s.Log != nil && decision == PrepareDecisionDegraded {
		s.Log.Warn("url batch returned no live delays; final rank uses TCP synthetic scores", "tcp_ok", len(tcpPings), "pool", len(pool), "telegram_target_circuit", circuitLabel(targetCircuitOpen), "event", "H4-degraded")
	}

	if s.Activity != nil {
		s.Activity(ctx, fmt.Sprintf("Ранжирование %d серверов…", len(pool)))
	}
	blOK := blOkMap(blPass, blEnabled)
	blUplinkOnly := blUplinkOnlyScope(blPass, blEnabled)
	tcpOnlyDegraded := decision == PrepareDecisionDegraded && !targetCircuitOpen
	var urlAliveKnown map[int64]struct{}
	if tcpOnlyDegraded && s.Store != nil {
		for _, id := range s.Store.GetBulkURLAlivePool(ctx) {
			if urlAliveKnown == nil {
				urlAliveKnown = make(map[int64]struct{})
			}
			urlAliveKnown[id] = struct{}{}
		}
	}
	ranked := rankProfiles(pool, tcpPings, urlDelays, blOK, blUplinkOnly, priorityIDs, s.isCooldown, tcpOnlyDegraded, urlAliveKnown)
	ranked = s.applyMultipathRank(ctx, ranked, tcpPings, urlDelays, priorityIDs, opts.WhitelistOnly)
	persistURLAlivePool(ctx, s.Store, urlDelays)
	maxPct := 100
	if s.Store != nil {
		maxPct = s.Store.EffectiveBuiltinFallbackMaxPct(ctx)
	}
	ranked = applyBuiltinFallbackCap(ranked, maxPct)
	ranked = capRankedForFallback(ctx, s.Store, ranked, tcpPings, urlDelays, blEnabled && blPass.FailClosed)
	if len(ranked) == 0 {
		return store.Profile{}, ResultAllDead, nil
	}
	best := ranked[0]
	ids := make([]int64, len(ranked))
	for i, p := range ranked {
		ids[i] = p.ID
	}
	_ = s.Store.SetFallbackQueue(ctx, ids)
	_ = s.Store.SetSelectedProxy(ctx, best.ID)
	_ = s.Store.SetLastKnownGood(ctx, best.ID)
	reason := fmt.Sprintf("live:best=%d queue=%d prepare_decision=%s tcp_ok=%d url_ok=%d telegram_target_circuit=%s degraded=%t%s", best.ID, len(ranked), decision, len(tcpPings), len(urlDelays), circuitLabel(targetCircuitOpen), decision == PrepareDecisionDegraded, blexit.ReasonSuffix(blPass, blEnabled))
	_ = s.Store.SetLastSelectReason(ctx, reason)
	s.Log.Info("queue prepared", "best", best.ID, "size", len(ranked), "event", "H4")
	return best, ResultSuccess, nil
}

func (s *Selector) setProbeEvidence(ev ProbeEvidence) {
	s.probeMu.Lock()
	s.probeEv = ev
	s.probeMu.Unlock()
}

func classifyPrepareDecision(tcpOK, urlOK int) PrepareDecision {
	if tcpOK == 0 {
		return PrepareDecisionHardDead
	}
	if urlOK == 0 {
		return PrepareDecisionDegraded
	}
	return PrepareDecisionOK
}

func (s *Selector) telegramCircuitOpen() bool {
	s.targetMu.Lock()
	defer s.targetMu.Unlock()
	if s.targetCircuitOpenUntil.IsZero() {
		return false
	}
	if time.Now().After(s.targetCircuitOpenUntil) {
		s.targetCircuitOpenUntil = time.Time{}
		return false
	}
	return true
}

func (s *Selector) recordTelegramBatch(attempted, succeeded int) {
	if attempted <= 0 {
		return
	}
	now := time.Now()
	s.targetMu.Lock()
	defer s.targetMu.Unlock()
	if s.targetWinStart.IsZero() || now.Sub(s.targetWinStart) > 3*time.Minute {
		s.targetWinStart = now
		s.targetURLAttempts = 0
		s.targetURLSuccesses = 0
	}
	s.targetURLAttempts += attempted
	s.targetURLSuccesses += succeeded
	if s.targetURLAttempts < 24 {
		return
	}
	ratio := float64(s.targetURLSuccesses) / float64(s.targetURLAttempts)
	if ratio < 0.15 {
		s.targetCircuitOpenUntil = now.Add(90 * time.Second)
		if s.Log != nil {
			s.Log.Warn("telegram target circuit opened", "attempted", s.targetURLAttempts, "succeeded", s.targetURLSuccesses, "ratio", ratio, "event", "H4-telegram-circuit")
		}
		return
	}
	if !s.targetCircuitOpenUntil.IsZero() && ratio > 0.35 {
		s.targetCircuitOpenUntil = time.Time{}
		if s.Log != nil {
			s.Log.Info("telegram target circuit closed", "attempted", s.targetURLAttempts, "succeeded", s.targetURLSuccesses, "ratio", ratio, "event", "H4-telegram-circuit")
		}
	}
}

func circuitLabel(open bool) string {
	if open {
		return "open"
	}
	return "closed"
}

func (s *Selector) LastProbeEvidence() ProbeEvidence {
	s.probeMu.Lock()
	defer s.probeMu.Unlock()
	return s.probeEv
}

func (s *Selector) TryMoveToFallback(ctx context.Context, currentID int64) (store.Profile, bool) {
	s.markProfileFailed(ctx, currentID)
	skip := func(id int64) bool { return s.shouldSkipFallback(ctx, id) }
	next, ok := s.Store.TryMoveFallbackSkip(ctx, currentID, skip)
	if !ok {
		return store.Profile{}, false
	}
	p, err := s.Store.ProfileByID(ctx, next)
	if err != nil {
		return store.Profile{}, false
	}
	return p, true
}

func (s *Selector) RecordFailure(id int64) {
	s.failures.Store(id, time.Now().UnixMilli())
}

func (s *Selector) newGen(owner Owner) int32 {
	if owner == OwnerAdapt || owner == OwnerSessionRecover {
		return s.adaptGen.Add(1)
	}
	return s.connectGen.Add(1)
}

func (s *Selector) stale(gen int32, owner Owner) bool {
	if gen < 0 {
		return false
	}
	if owner == OwnerAdapt || owner == OwnerSessionRecover {
		return gen != s.adaptGen.Load()
	}
	return gen != s.connectGen.Load()
}

// coldPrepareProbes runs TCP on the pool; with picker available, URL batch on top-K runs in parallel.
func (s *Selector) coldPrepareProbes(ctx context.Context, pool []store.Profile, opts PrepareOpts, priority map[int64]struct{}, gen int32) (map[int64]int, map[int64]int) {
	if opts.Tester != nil && len(opts.ProxyNames) > 0 {
		tcp := s.tcpProbeAll(ctx, pool, priority, opts.WhitelistOnly, gen, opts.Owner)
		return tcp, s.urlTestTop(ctx, pool, opts, tcp, priority)
	}
	usePicker := s.Ephemeral != nil && pickerReady(ctx, s.Ephemeral.Picker)
	if !usePicker {
		tcp := s.tcpProbeAll(ctx, pool, priority, opts.WhitelistOnly, gen, opts.Owner)
		url := map[int64]int{}
		if s.Ephemeral != nil {
			if s.Activity != nil {
				s.Activity(ctx, "Тест серверов (URL, batch)…")
			}
			url = s.ephemeralURLTest(ctx, pool, tcp, priority, gen, opts.Owner)
		}
		return tcp, url
	}
	pipeline := urlTestCandidates(pool, nil, priority, urlTestCapPicker)
	if s.Activity != nil && len(pipeline) > 0 {
		s.Activity(ctx, fmt.Sprintf("TCP + Picker %d (параллельно)…", len(pipeline)))
	}
	var tcpPings map[int64]int
	var urlDelays map[int64]int
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		tcpPings = s.tcpProbeAll(ctx, pool, priority, opts.WhitelistOnly, gen, opts.Owner)
	}()
	go func() {
		defer wg.Done()
		if len(pipeline) == 0 {
			return
		}
		urlDelays = s.Ephemeral.testProfilesPickerWithCatchup(ctx, pipeline)
	}()
	wg.Wait()
	if s.stale(gen, opts.Owner) {
		return nil, nil
	}
	if len(urlDelays) == 0 {
		sorted := urlTestCandidates(pool, tcpPings, priority, urlTestCapPicker)
		if len(sorted) > 0 {
			urlDelays = s.Ephemeral.testProfilesPickerWithCatchup(ctx, sorted)
		}
	}
	return tcpPings, urlDelays
}

func (s *Selector) isCooldown(id int64) bool {
	v, ok := s.failures.Load(id)
	if !ok {
		return false
	}
	at := v.(int64)
	if time.Now().UnixMilli()-at >= profileFailureCooldownMs {
		s.failures.Delete(id)
		return false
	}
	return true
}

func (s *Selector) buildPool(ctx context.Context, all []store.Profile, wlOnly bool, handoffIDs map[int64]struct{}) ([]store.Profile, map[int64]struct{}) {
	priority := map[int64]struct{}{}
	for id := range handoffIDs {
		priority[id] = struct{}{}
	}
	if wlOnly {
		wl, _ := bootstrap.WLPoolProfiles(ctx, s.Store)
		seen := map[int64]struct{}{}
		var head, rest []store.Profile
		for _, p := range wl {
			priority[p.ID] = struct{}{}
			head = append(head, p)
			seen[p.ID] = struct{}{}
		}
		for _, p := range all {
			if _, ok := seen[p.ID]; ok {
				continue
			}
			if p.IsSubscriptionWhitelistMarked() {
				priority[p.ID] = struct{}{}
				head = append(head, p)
			} else {
				rest = append(rest, p)
			}
		}
		return append(head, rest...), priority
	}
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
	return subs, priority
}

func (s *Selector) ephemeralURLTest(ctx context.Context, pool []store.Profile, tcp map[int64]int, priority map[int64]struct{}, gen int32, owner Owner) map[int64]int {
	cap := urlTestCapDefault
	if s.Ephemeral != nil && pickerReady(ctx, s.Ephemeral.Picker) {
		cap = urlTestCapPicker
	}
	sorted := urlTestCandidates(pool, tcp, priority, cap)
	if ctx.Err() != nil || s.stale(gen, owner) {
		return nil
	}
	if s.Activity != nil && (s.Ephemeral == nil || !pickerReady(ctx, s.Ephemeral.Picker)) {
		s.Activity(ctx, fmt.Sprintf("URL тест %d серверов (лучшие по TCP)…", len(sorted)))
	}
	out := s.Ephemeral.TestProfilesBatch(ctx, sorted)
	ok := len(out)
	if s.Log != nil {
		s.Log.Info("ephemeral pretest batch", "ok", ok, "fail", len(sorted)-ok, "candidates", len(sorted))
	}
	return out
}

func (s *Selector) handoffPriority(ctx context.Context, selected int64, handoff bool) map[int64]struct{} {
	out := map[int64]struct{}{}
	if !handoff {
		return out
	}
	if selected > 0 {
		out[selected] = struct{}{}
	}
	if lg, _ := s.Store.LastKnownGood(ctx); lg > 0 {
		out[lg] = struct{}{}
	}
	if q, _ := s.Store.FallbackQueue(ctx); len(q) > 0 {
		for i, id := range q {
			if i >= 12 {
				break
			}
			out[id] = struct{}{}
		}
	}
	return out
}

func (s *Selector) filterCemeteryFromPool(ctx context.Context, pool []store.Profile, priority map[int64]struct{}) []store.Profile {
	if s.Store == nil || len(pool) == 0 {
		return pool
	}
	out := make([]store.Profile, 0, len(pool))
	for _, p := range pool {
		if _, pri := priority[p.ID]; pri {
			out = append(out, p)
			continue
		}
		meta, err := s.Store.ProbeMetaByID(ctx, p.ID)
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

func (s *Selector) tcpProbeAll(ctx context.Context, profiles []store.Profile, priority map[int64]struct{}, wl bool, gen int32, owner Owner) map[int64]int {
	cap := openNetTCPProbeCap
	if wl {
		cap = 128
	}
	targets := profiles
	if len(targets) > cap {
		targets = compactPool(targets, priority, cap)
	}
	timeout := s.TCPTimeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	out := make(map[int64]int)
	var mu sync.Mutex
	var done atomic.Int32
	total := len(targets)
	if s.Activity != nil && total > 0 {
		s.Activity(ctx, fmt.Sprintf("TCP тест 0/%d", total))
	}
	sem := make(chan struct{}, tcpProbeWorkers)
	var wg sync.WaitGroup
	for _, p := range targets {
		if ctx.Err() != nil || s.stale(gen, owner) {
			break
		}
		host, port, err := profileHostPort(p)
		if err != nil || host == "" {
			continue
		}
		p, host, port := p, host, port
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if ctx.Err() != nil || s.stale(gen, owner) {
				return
			}
			start := time.Now()
			dialer := net.Dialer{Timeout: timeout}
			c, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
			n := done.Add(1)
			if s.Activity != nil && (n == int32(total) || n%8 == 0) {
				s.Activity(ctx, fmt.Sprintf("TCP тест %d/%d", n, total))
			}
			if err != nil {
				return
			}
			_ = c.Close()
			ms := int(time.Since(start).Milliseconds())
			mu.Lock()
			out[p.ID] = ms
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

func (s *Selector) urlTestTop(ctx context.Context, pool []store.Profile, opts PrepareOpts, tcp map[int64]int, priority map[int64]struct{}) map[int64]int {
	capN := urlTestCapDefault
	if opts.NetworkHandoff {
		capN = urlTestCapHandoff
	}
	sorted := urlTestCandidates(pool, tcp, priority, capN)
	if len(sorted) == 0 || opts.Tester == nil {
		return nil
	}
	out := make(map[int64]int)
	var mu sync.Mutex
	sem := make(chan struct{}, urlTestWorkers)
	var wg sync.WaitGroup
	for _, p := range sorted {
		name, ok := opts.ProxyNames[p.ID]
		if !ok || name == "" {
			continue
		}
		p, name := p, name
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			delay, err := opts.Tester.TestProxyDelay(ctx, name)
			if err != nil || delay <= 0 {
				return
			}
			mu.Lock()
			out[p.ID] = delay
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

// urlTestCandidates picks profiles for objective URL delay test: handoff/priority first, then TCP-live, ranked by TCP.
func urlTestCandidates(pool []store.Profile, tcp map[int64]int, priority map[int64]struct{}, cap int) []store.Profile {
	if cap <= 0 || len(pool) == 0 {
		return nil
	}
	seen := make(map[int64]struct{})
	var cand []store.Profile
	add := func(p store.Profile) {
		if _, ok := seen[p.ID]; ok {
			return
		}
		seen[p.ID] = struct{}{}
		cand = append(cand, p)
	}
	for _, p := range pool {
		if _, pri := priority[p.ID]; pri {
			add(p)
		}
	}
	for _, p := range pool {
		if tcp[p.ID] > 0 {
			add(p)
		}
	}
	if len(cand) == 0 {
		cand = append(cand, pool...)
	}
	sorted := rankProfiles(cand, tcp, nil, nil, false, priority, func(int64) bool { return false }, false, nil)
	if len(sorted) > cap {
		sorted = sorted[:cap]
	}
	return sorted
}

func rankProfiles(pool []store.Profile, tcp, url map[int64]int, blOK map[int64]bool, blUplinkOnly bool, priority map[int64]struct{}, cooldown func(int64) bool, tcpOnlyDegraded bool, urlAliveKnown map[int64]struct{}) []store.Profile {
	out := append([]store.Profile(nil), pool...)
	sort.SliceStable(out, func(i, j int) bool {
		return lessProfile(out[i], out[j], tcp, url, blOK, blUplinkOnly, priority, cooldown, tcpOnlyDegraded, urlAliveKnown)
	})
	return out
}

func lessProfile(a, b store.Profile, tcp, url map[int64]int, blOK map[int64]bool, blUplinkOnly bool, priority map[int64]struct{}, cooldown func(int64) bool, tcpOnlyDegraded bool, urlAliveKnown map[int64]struct{}) bool {
	if cooldown(a.ID) != cooldown(b.ID) {
		return cooldown(b.ID)
	}
	sa, sb := compositeScore(a, tcp, url, blOK, blUplinkOnly, tcpOnlyDegraded, urlAliveKnown), compositeScore(b, tcp, url, blOK, blUplinkOnly, tcpOnlyDegraded, urlAliveKnown)
	if sa != sb {
		return sa < sb
	}
	_, pa := priority[a.ID]
	_, pb := priority[b.ID]
	if pa != pb {
		return pa
	}
	return a.UserOrder < b.UserOrder
}

const tcpOnlyDegradedPenalty = 1 << 25

func compositeScore(p store.Profile, tcp, url map[int64]int, blOK map[int64]bool, blUplinkOnly bool, tcpOnlyDegraded bool, urlAliveKnown map[int64]struct{}) int {
	if blOK != nil {
		if blUplinkOnly {
			if profileclass.IsBLModeUplink(p) && !blOK[p.ID] {
				return 1 << 30
			}
		} else if !blOK[p.ID] {
			return 1 << 30
		}
	}
	u := url[p.ID]
	if u > 0 {
		return u
	}
	t := tcp[p.ID]
	if t > 0 {
		syn := t * 3
		if syn < 40 {
			syn = 40
		}
		if syn > 900 {
			syn = 900
		}
		score := 10*t + syn
		if tcpOnlyDegraded {
			if len(urlAliveKnown) == 0 {
				score += tcpOnlyDegradedPenalty
			} else if _, ok := urlAliveKnown[p.ID]; !ok {
				score += tcpOnlyDegradedPenalty
			}
		}
		return score
	}
	return 1 << 30
}

func compactPool(profiles []store.Profile, priority map[int64]struct{}, max int) []store.Profile {
	var pri, rest []store.Profile
	for _, p := range profiles {
		if _, ok := priority[p.ID]; ok {
			pri = append(pri, p)
		} else {
			rest = append(rest, p)
		}
	}
	out := append(pri, rest...)
	if len(out) > max {
		out = out[:max]
	}
	return out
}

func profileHostPort(p store.Profile) (host string, port int, err error) {
	switch p.Type {
	case "vless", "":
		v, e := configgen.ParseVLESSURI(p.URI)
		return v.Server, v.Port, e
	case "trojan":
		t, e := configgen.ParseTrojanURI(p.URI)
		return t.Server, t.Port, e
	case "hysteria", "hysteria2":
		h, e := configgen.ParseHysteriaURI(p.URI)
		return h.Server, h.Port, e
	default:
		return "", 0, fmt.Errorf("no tcp probe for %s", p.Type)
	}
}

func filterEnabled(in []store.Profile) []store.Profile {
	var out []store.Profile
	for _, p := range in {
		if p.Enabled {
			out = append(out, p)
		}
	}
	return out
}

// needWLBuiltin syncs trojan pool when restricted network or user opted into builtin rescue.
func (s *Selector) needWLBuiltin(ctx context.Context, whitelistOnly bool) bool {
	if whitelistOnly {
		return true
	}
	return s.Store != nil && s.Store.WLBuiltinConnectEnabled(ctx)
}
