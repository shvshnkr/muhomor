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

	"github.com/muhomor/muhomor/internal/bootstrap"
	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/activity"
	"github.com/muhomor/muhomor/internal/store"
)

const (
	openNetTCPProbeCap       = 128
	profileFailureCooldownMs = 30 * 60 * 1000
	urlTestCapHandoff        = 12
	urlTestCapDefault        = 70 // URL batch on top TCP survivors (Dahusim-scale; speed via one mihomo + parallel /delay)
	tcpProbeWorkers          = 48
	urlTestWorkers           = 24
)

// Owner mirrors PrepareOwner in Kotlin.
type Owner int

const (
	OwnerConnect Owner = iota
	OwnerAdapt
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
}

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
	priority := s.handoffPriority(ctx, selectedBefore, opts.NetworkHandoff)
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
	tcpPings := s.tcpProbeAll(ctx, pool, priorityIDs, opts.WhitelistOnly, gen, opts.Owner)
	urlDelays := map[int64]int{}
	if opts.Tester != nil && len(opts.ProxyNames) > 0 {
		urlDelays = s.urlTestTop(ctx, pool, opts, tcpPings, priorityIDs)
	} else if s.Ephemeral != nil {
		if s.Activity != nil {
			s.Activity(ctx, "Тест серверов (URL, batch)…")
		}
		urlDelays = s.ephemeralURLTest(ctx, pool, tcpPings, priorityIDs, gen, opts.Owner)
	}

	if s.stale(gen, opts.Owner) {
		return store.Profile{}, ResultNoProfiles, context.Canceled
	}

	if len(tcpPings) == 0 && len(urlDelays) == 0 && len(pool) > 0 {
		s.Log.Warn("all probes dead", "count", len(pool), "tcp", 0, "url", 0, "event", "H22")
		return store.Profile{}, ResultAllDead, nil
	}
	if s.Log != nil && len(urlDelays) == 0 && len(tcpPings) > 0 {
		s.Log.Warn("url batch returned no live delays; final rank uses TCP synthetic scores", "tcp_ok", len(tcpPings), "pool", len(pool), "event", "H4-degraded")
	}

	if s.Activity != nil {
		s.Activity(ctx, fmt.Sprintf("Ранжирование %d серверов…", len(pool)))
	}
	ranked := rankProfiles(pool, tcpPings, urlDelays, priorityIDs, s.isCooldown)
	ranked = s.applyMultipathRank(ctx, ranked, tcpPings, urlDelays, priorityIDs, opts.WhitelistOnly)
	persistURLAlivePool(ctx, s.Store, urlDelays)
	maxPct := 100
	if s.Store != nil {
		maxPct = s.Store.EffectiveBuiltinFallbackMaxPct(ctx)
	}
	ranked = applyBuiltinFallbackCap(ranked, maxPct)
	ranked = capRankedForFallback(ranked, tcpPings, urlDelays)
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
	reason := fmt.Sprintf("live:best=%d queue=%d tcp_ok=%d url_ok=%d", best.ID, len(ranked), len(tcpPings), len(urlDelays))
	_ = s.Store.SetLastSelectReason(ctx, reason)
	s.Log.Info("queue prepared", "best", best.ID, "size", len(ranked), "event", "H4")
	return best, ResultSuccess, nil
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
	if owner == OwnerAdapt {
		return s.adaptGen.Add(1)
	}
	return s.connectGen.Add(1)
}

func (s *Selector) stale(gen int32, owner Owner) bool {
	if gen < 0 {
		return false
	}
	if owner == OwnerAdapt {
		return gen != s.adaptGen.Load()
	}
	return gen != s.connectGen.Load()
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
	sorted := urlTestCandidates(pool, tcp, priority, urlTestCapDefault)
	if ctx.Err() != nil || s.stale(gen, owner) {
		return nil
	}
	if s.Activity != nil {
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
	sorted := rankProfiles(cand, tcp, nil, priority, func(int64) bool { return false })
	if len(sorted) > cap {
		sorted = sorted[:cap]
	}
	return sorted
}

func rankProfiles(pool []store.Profile, tcp, url map[int64]int, priority map[int64]struct{}, cooldown func(int64) bool) []store.Profile {
	out := append([]store.Profile(nil), pool...)
	sort.SliceStable(out, func(i, j int) bool {
		return lessProfile(out[i], out[j], tcp, url, priority, cooldown)
	})
	return out
}

func lessProfile(a, b store.Profile, tcp, url map[int64]int, priority map[int64]struct{}, cooldown func(int64) bool) bool {
	if cooldown(a.ID) != cooldown(b.ID) {
		return cooldown(b.ID)
	}
	sa, sb := compositeScore(a, tcp, url), compositeScore(b, tcp, url)
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

func compositeScore(p store.Profile, tcp, url map[int64]int) int {
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
		return 10*t + syn
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
