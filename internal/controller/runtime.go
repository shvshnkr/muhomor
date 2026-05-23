package controller

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/muhomor/muhomor/internal/aggregate"
	"github.com/muhomor/muhomor/internal/api"
	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/mihomo"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/probe"
	"github.com/muhomor/muhomor/internal/reachability"
	"github.com/muhomor/muhomor/internal/routing"
	"github.com/muhomor/muhomor/internal/selector"
	"github.com/muhomor/muhomor/internal/simplemode"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

// Runtime coordinates store, mihomo, and simple mode (Phase 2).
type Runtime struct {
	Paths  paths.Layout
	Store  *store.Store
	Log    *slog.Logger
	mu     sync.Mutex
	status Status
	mihomo *mihomo.Client
	build  configgen.BuildOptions
	proxy  string

	connect       *simplemode.Connector
	selector      *selector.Selector
	adaptor       *simplemode.Adaptor
	health        *simplemode.SessionHealth
	maintenance   *simplemode.Maintenance
	netmon        *simplemode.NetworkMonitor
	reachCache    simplemode.ReachabilityCache
	daemonCtx     context.Context
	Events        *EventHub
	connectMu     sync.Mutex
	connectCancel context.CancelFunc
}

// ApplyCLISettings merges daemon flags into store (DesktopMain --proxy-port etc.).
func ApplyCLISettings(ctx context.Context, st *store.Store, serviceMode string, mixedPort int, proxyAuth string, routeQuick int, tun bool) error {
	set, err := st.LoadSettings(ctx)
	if err != nil {
		return err
	}
	if serviceMode != "" {
		set.ServiceMode = serviceMode
		set.TunEnable = serviceMode == store.ServiceModeVPN
	}
	if mixedPort > 0 {
		set.MixedPort = mixedPort
	}
	if proxyAuth != "" {
		if proxyAuth == "none" {
			set.InboundUser, set.InboundPassword = "", ""
		} else if i := stringsIndex(proxyAuth, ':'); i > 0 {
			set.InboundUser = proxyAuth[:i]
			set.InboundPassword = proxyAuth[i+1:]
		}
	}
	if routeQuick >= 0 {
		set.RouteQuickProfile = routeQuick
		_ = st.SetRouteQuickProfile(ctx, routeQuick)
	}
	if tun {
		set.TunEnable = true
		set.ServiceMode = store.ServiceModeVPN
	}
	return st.SaveSettings(ctx, set)
}

func stringsIndex(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func NewRuntime(layout paths.Layout, st *store.Store, log *slog.Logger) *Runtime {
	if log == nil {
		log = slog.Default()
	}
	sel := &selector.Selector{Store: st, Log: log}
	r := &Runtime{
		Paths:       layout,
		Store:       st,
		Log:         log,
		status:      Status{State: StateIdle},
		Events:      NewEventHub(),
		build:       configgen.DefaultBuildOptions(),
		selector:    sel,
		maintenance: &simplemode.Maintenance{Store: st, Updater: newSubscriptionUpdater(st), Log: log},
	}
	sel.Ephemeral = &selector.EphemeralTester{MihomoBin: mihomo.ResolveBin(), Store: st, Log: log, Activity: r.setActivity}
	r.connect = &simplemode.Connector{
		Store:     st,
		Selector:  sel,
		Probe:     r.probeFresh,
		StartFn:   r.startProfile,
		Bootstrap: true,
		Updater:   newSubscriptionUpdater(st),
		Log:       log,
		Activity:  r.setActivity,
	}
	sel.Activity = r.setActivity
	r.adaptor = &simplemode.Adaptor{
		Store:    st,
		Selector: sel,
		Probe:    r.probeFresh,
		Reselect: r.reselectProfile,
		Log:      log,
		WLRescue: func(ctx context.Context) bool {
			return st.WLBuiltinConnectEnabled(ctx)
		},
	}
	r.health = &simplemode.SessionHealth{
		Selector: sel,
		Store:    st,
		Feedback: &aggregate.Feedback{Store: st},
		OnUnhealthy: func(ctx context.Context, id int64) error {
			next, ok := sel.TryMoveToFallback(ctx, id)
			if !ok {
				sel.RecordFailure(id)
				return fmt.Errorf("no fallback")
			}
			probe, _ := r.cachedProbe(ctx)
			return r.startProfile(ctx, next, probe)
		},
		Log: log,
	}
	r.netmon = &simplemode.NetworkMonitor{
		OnHandoff: func(ctx context.Context, reason string) {
			r.reachCache.Invalidate()
			r.adaptor.ScheduleAdaptation(ctx, reason)
		},
	}
	r.refreshBuildOptions(context.Background())
	return r
}

func (r *Runtime) refreshBuildOptions(ctx context.Context) {
	set, err := r.Store.LoadSettings(ctx)
	if err != nil {
		return
	}
	_ = r.Store.ClearAutoInboundCredentials(ctx)
	set, _ = r.Store.LoadSettings(ctx)
	r.build = configgen.OptionsFromSettings(set, r.build)
	if set.ServiceMode == store.ServiceModeVPN {
		r.build.Tun.Enable = true
	}
}

func (r *Runtime) SetDaemonContext(ctx context.Context) {
	r.daemonCtx = ctx
	r.netmon.Start(ctx)
	rulesDir := r.Paths.MihomoDir() + string(os.PathSeparator) + "ruleset"
	sched := &Scheduler{
		Store:  r.Store,
		Assets: &subscription.AssetUpdater{RulesDir: rulesDir},
		Subs:   r.connect.Updater,
		Log:    r.Log,
	}
	go sched.Run(ctx)
	if st := r.Store; st != nil && st.ProbeSchedulerEnabled(ctx) {
		ps := &probe.Scheduler{Store: st, Log: r.Log, Config: probe.ConfigFromStore(ctx, st)}
		go ps.Run(ctx)
	}
}

// StartChain connects using relay chain (Phase 3).
func (r *Runtime) StartChain(ctx context.Context, profileIDs []int64) error {
	if len(profileIDs) == 0 {
		return fmt.Errorf("empty chain")
	}
	var members []store.Profile
	for _, id := range profileIDs {
		p, err := r.Store.ProfileByID(ctx, id)
		if err != nil {
			return fmt.Errorf("profile %d: %w", id, err)
		}
		members = append(members, p)
	}
	r.refreshBuildOptions(ctx)
	qp, _ := r.Store.RouteQuickProfile(ctx)
	rules := routing.QuickProfileRuleLines(qp)
	rulesDir := r.Paths.MihomoDir() + string(os.PathSeparator) + "ruleset"
	yaml, proxyName, err := configgen.BuildChainConfig(configgen.ChainSpec{
		Name:    "user-chain",
		Members: members,
	}, r.build, rules, rulesDir)
	if err != nil {
		return err
	}
	cfgPath := r.Paths.ConfigPath()
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		return err
	}
	if _, err := r.startMihomoClient(ctx, cfgPath); err != nil {
		return err
	}
	r.proxy = proxyName
	if set, err := r.Store.LoadSettings(ctx); err == nil {
		set.ChainProfileIDs = profileIDs
		_ = r.Store.SaveSettings(ctx, set)
	}
	last := members[len(members)-1]
	_ = r.Store.SetCurrentProfileID(ctx, last.ID)
	r.setStatus(Status{State: StateConnected, Connected: true, ProfileID: last.ID, ProfileName: last.Name, ProxyName: proxyName})
	r.Log.Info("chain connected", "hops", len(members), "exit", proxyName, "event", "H30")
	return nil
}

func (r *Runtime) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

func (r *Runtime) setStatus(s Status) {
	r.mu.Lock()
	r.status = s
	r.mu.Unlock()
	r.publishStatusEvent("status")
}

// ConnectProfile starts a specific profile (expert / API).
func (r *Runtime) ConnectProfile(ctx context.Context, profileID int64) error {
	p, err := r.Store.ProfileByID(ctx, profileID)
	if err != nil {
		return err
	}
	if !p.Enabled {
		return fmt.Errorf("profile %d is disabled", profileID)
	}
	probe, _ := r.cachedProbe(ctx)
	return r.startProfile(ctx, p, probe)
}

// Ping tests active proxy delay via mihomo when connected.
func (r *Runtime) Ping(ctx context.Context) (api.PingResponse, error) {
	st := r.Status()
	out := api.PingResponse{
		Timestamp:   time.Now().UnixMilli(),
		Connected:   st.ConnectedBool(),
		ProfileName: st.ProfileName,
		ProxyName:   st.ProxyName,
	}
	r.mu.Lock()
	client := r.mihomo
	proxy := r.proxy
	r.mu.Unlock()
	if client != nil && proxy != "" {
		d, err := client.TestProxyDelay(ctx, proxy)
		out.DelayMs = d
		if err != nil {
			out.Error = err.Error()
		}
	}
	pingPath := r.Paths.CacheDir + string(os.PathSeparator) + "desktop-control-ping.txt"
	_ = os.MkdirAll(r.Paths.CacheDir, 0o700)
	body := fmt.Sprintf("timestamp=%d\nconnected=%t\ndelay_ms=%d\nprofile=%s\nproxy=%s\n",
		out.Timestamp, out.Connected, out.DelayMs, out.ProfileName, out.ProxyName)
	_ = os.WriteFile(pingPath, []byte(body), 0o644)
	out.Path = pingPath
	return out, nil
}

func (r *Runtime) beginConnect(ctx context.Context) context.Context {
	r.connectMu.Lock()
	defer r.connectMu.Unlock()
	if r.connectCancel != nil {
		r.connectCancel()
	}
	ctx, cancel := context.WithCancel(ctx)
	r.connectCancel = cancel
	return ctx
}

func (r *Runtime) endConnect() {
	r.connectMu.Lock()
	defer r.connectMu.Unlock()
	if r.connectCancel != nil {
		r.connectCancel()
		r.connectCancel = nil
	}
}

func (r *Runtime) cancelInFlightConnect() {
	r.connectMu.Lock()
	cancel := r.connectCancel
	r.connectCancel = nil
	r.connectMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (r *Runtime) Start(_ context.Context) error {
	// Long connect must not use HTTP request ctx (client disconnect / GUI cancel would abort probes).
	ctx := r.beginConnect(context.Background())
	defer r.endConnect()
	r.setStatus(Status{State: StateConnecting})
	r.setActivity(ctx, "Подключение…")
	err := r.connect.Connect(ctx)
	if err != nil {
		if ctx.Err() != nil {
			r.clearActivity(context.Background())
			r.setStatus(Status{State: StateStopped})
			return fmt.Errorf("connect aborted")
		}
		r.setActivity(ctx, err.Error())
		r.setStatus(Status{State: StateIdle})
		return err
	}
	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	r.cancelInFlightConnect()
	r.selector.CancelConnect()
	if r.adaptor != nil {
		r.adaptor.CancelAll()
	}
	r.reachCache.Invalidate()
	r.health.Stop()
	r.clearActivity(ctx)
	r.setStatus(Status{State: StateStopping})
	r.mu.Lock()
	client := r.mihomo
	r.mihomo = nil
	r.mu.Unlock()
	if client != nil {
		client.Stop()
	}
	mihomo.KillAll()
	r.setStatus(Status{State: StateStopped})
	return nil
}

func (r *Runtime) Reload(ctx context.Context) error {
	r.mu.Lock()
	client := r.mihomo
	proxy := r.proxy
	r.mu.Unlock()
	if client == nil {
		return nil
	}
	if proxy != "" {
		if d, err := client.TestProxyDelay(ctx, proxy); err == nil && d > 0 {
			return client.Reload(ctx)
		}
	}
	return r.ReapplyCurrentProfile(ctx)
}

// ReapplyCurrentProfile rebuilds YAML for the active profile (settings/route change, no selector).
func (r *Runtime) ReapplyCurrentProfile(ctx context.Context) error {
	id, _ := r.Store.CurrentProfileID(ctx)
	if id <= 0 {
		r.mu.Lock()
		client := r.mihomo
		r.mu.Unlock()
		if client != nil {
			return client.Reload(ctx)
		}
		return nil
	}
	p, err := r.Store.ProfileByID(ctx, id)
	if err != nil {
		return err
	}
	probe, _ := r.cachedProbe(ctx)
	return r.startProfileAttempt(ctx, p, probe)
}

func (r *Runtime) Adapt(ctx context.Context, reason string) {
	r.adaptor.ScheduleAdaptation(ctx, reason)
	if r.Events != nil {
		st := r.statusSnapshot(ctx)
		r.Events.Publish(api.Event{Type: "adapt", Reason: reason, Status: &st})
	}
}

func (r *Runtime) reselectProfile(ctx context.Context, p store.Profile, _ bool) error {
	probe, _ := r.cachedProbe(ctx)
	return r.startProfile(ctx, p, probe)
}

func (r *Runtime) probeFresh(ctx context.Context, fast bool) reachability.Result {
	res := reachability.Probe(ctx, fast)
	r.reachCache.Put(res, 30*time.Second)
	return res
}

func (r *Runtime) cachedProbe(ctx context.Context) (reachability.Result, bool) {
	if res, ok := r.reachCache.Get(); ok {
		return res, true
	}
	res := r.probeFresh(ctx, true)
	return res, false
}

func (r *Runtime) startProfile(ctx context.Context, profile store.Profile, probe reachability.Result) error {
	const maxPostConnectSwitch = 12
	for attempt := 0; attempt < maxPostConnectSwitch; attempt++ {
		if err := r.startProfileAttempt(ctx, profile, probe); err != nil {
			if attempt+1 >= maxPostConnectSwitch {
				return err
			}
			next, ok := r.selector.TryMoveToFallback(ctx, profile.ID)
			if !ok {
				return err
			}
			r.setActivity(ctx, "Сервер нестабилен, переключение…")
			profile = next
			continue
		}
		return nil
	}
	return fmt.Errorf("post-connect: fallbacks exhausted")
}

func (r *Runtime) startProfileAttempt(ctx context.Context, profile store.Profile, probe reachability.Result) error {
	r.refreshBuildOptions(ctx)
	r.setActivity(ctx, "Запуск mihomo…")
	r.setStatus(Status{State: StateConnecting, ProfileID: profile.ID, ProfileName: profile.Name})
	set, _ := r.Store.LoadSettings(ctx)
	qp, _ := r.Store.RouteQuickProfile(ctx)
	rules := routing.QuickProfileRuleLines(qp)
	rules = routing.ApplyRuGeoRules(ctx, r.Store, profile.ID, rules)
	rulesDir := r.Paths.MihomoDir() + string(os.PathSeparator) + "ruleset"
	poolLegs := r.loadBulkPoolLegs(ctx, profile.ID)
	urlAlive := r.Store.GetBulkURLAlivePool(ctx)
	if set.AggregationMode == store.AggregationModeFlowAggregate {
		profile, poolLegs = filterProfilesForBulk(profile, poolLegs, urlAlive)
	}
	var yaml string
	var proxyName string
	var plan configgen.BulkPlan
	var err error
	if set.AggregationMode == store.AggregationModeFlowAggregate {
		yaml, proxyName, plan, err = configgen.BuildFromProfileBulk(profile, poolLegs, r.build, rules, rulesDir, qp, set)
	} else {
		yaml, proxyName, err = configgen.BuildFromProfileExtQP(profile, r.build, rules, rulesDir, qp)
		plan = configgen.BulkPlan{MatchTarget: configgen.GroupPROXY, FallbackReason: "legacy_mode"}
	}
	if err != nil {
		_ = r.Store.SetBulkStatus(ctx, store.BulkStatus{
			AggregationMode:    set.AggregationMode,
			BulkEnabled:        set.BulkEnabled,
			BulkActive:         false,
			BulkFallbackReason: "config_error",
			BulkMinLegs:        set.BulkMinHealthyLegs,
			BulkMaxLegs:        r.Store.EffectiveBulkMaxLegs(ctx),
			BulkMemberTags:     nil,
		})
		return err
	}
	_ = r.Store.SetBulkStatus(ctx, store.BulkStatus{
		AggregationMode:    set.AggregationMode,
		BulkEnabled:        set.BulkEnabled,
		BulkActive:         plan.Active,
		BulkMemberCount:    len(plan.BulkTags),
		BulkMemberTags:     append([]string(nil), plan.BulkTags...),
		BulkMinLegs:        set.BulkMinHealthyLegs,
		BulkMaxLegs:        r.Store.EffectiveBulkMaxLegs(ctx),
		BulkFallbackReason: plan.FallbackReason,
	})
	mpPool := len(r.Store.GetMultipathChannelPool(ctx))
	if plan.Active {
		r.Log.Info("bulk data-plane active", "members", len(plan.BulkTags), "match", plan.MatchTarget, "mp_pool", mpPool, "event", "BULK-active")
	} else if set.AggregationMode == store.AggregationModeFlowAggregate {
		r.Log.Info("bulk fallback", "reason", plan.FallbackReason, "mp_pool", mpPool, "rendered", len(plan.BulkTags), "pool_legs", len(poolLegs), "event", "BULK-fallback")
	}
	// #region agent log
	agentDebugLog("H1,H2,H3", "internal/controller/runtime.go:startProfileAttempt:bulk-plan", "bulk plan before mihomo start", map[string]any{
		"profile_id":       profile.ID,
		"aggregation_mode": set.AggregationMode,
		"bulk_enabled":     set.BulkEnabled,
		"plan_active":      plan.Active,
		"match_target":     plan.MatchTarget,
		"fallback_reason":  plan.FallbackReason,
		"bulk_tags":        plan.BulkTags,
		"primary_proxy":    proxyName,
	})
	// #endregion
	cfgPath := r.Paths.ConfigPath()
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		return err
	}
	client, err := r.startMihomoClient(ctx, cfgPath)
	if err != nil {
		return err
	}
	effectiveProxy := proxyName
	if plan.Active && plan.MatchTarget != "" {
		effectiveProxy = plan.MatchTarget
	}
	r.proxy = effectiveProxy
	// #region agent log
	agentDebugLog("H1,H3,H4", "internal/controller/runtime.go:startProfileAttempt:effective-proxy", "effective proxy selected for post-connect and runtime", map[string]any{
		"profile_id":      profile.ID,
		"effective_proxy": effectiveProxy,
		"primary_proxy":   proxyName,
		"bulk_active":     plan.Active,
		"bulk_members":    len(plan.BulkTags),
	})
	// #endregion

	r.setActivity(ctx, fmt.Sprintf("Проверка соединения через %s…", effectiveProxy))
	time.Sleep(800 * time.Millisecond)
	testURL := r.Store.ConnectionTestURL(ctx)
	testMs := r.Store.ConnectionTestTimeoutMs(ctx)
	if testMs < 8000 {
		testMs = 8000
	}
	_, delay, delayErr := postConnectDelay(ctx, client, plan, proxyName, testURL, testMs)
	// #region agent log
	delayErrText := ""
	if delayErr != nil {
		delayErrText = delayErr.Error()
	}
	agentDebugLog("H1,H4", "internal/controller/runtime.go:startProfileAttempt:post-connect-delay", "post-connect delay result", map[string]any{
		"profile_id":      profile.ID,
		"effective_proxy": effectiveProxy,
		"primary_proxy":   proxyName,
		"delay_ms":        delay,
		"err":             delayErrText,
		"test_url":        testURL,
		"timeout_ms":      testMs,
	})
	// #endregion
	if delayErr != nil || delay <= 0 {
		r.mu.Lock()
		if r.mihomo != nil {
			r.mihomo.Stop()
			r.mihomo = nil
		}
		r.mu.Unlock()
		mihomo.KillAll()
		r.Log.Info("post-connect url test failed", "profile", profile.ID, "proxy", effectiveProxy, "primary_proxy", proxyName, "delay", delay, "err", delayErr, "event", "H3")
		if delayErr != nil {
			return fmt.Errorf("post-connect url test failed: %w", delayErr)
		}
		return fmt.Errorf("post-connect url test failed: no response via proxy")
	}
	r.Log.Info("post-connect url test ok", "profile", profile.ID, "proxy", effectiveProxy, "delay_ms", delay, "event", "H3")

	r.health.Delay = client
	memberTags := plan.BulkTags
	if !plan.Active {
		memberTags = nil
	}
	r.health.Start(r.daemonCtx, profile.ID, effectiveProxy, memberTags)

	_ = r.Store.SetCurrentProfileID(ctx, profile.ID)
	r.clearActivity(ctx)
	r.setStatus(Status{
		State:       StateConnected,
		Connected:   true,
		ProfileID:   profile.ID,
		ProfileName: profile.Name,
		ProxyName:   effectiveProxy,
	})
	r.Log.Info("service connected", "profile", profile.Name, "proxy", effectiveProxy, "primary_proxy", proxyName, "quick_profile", qp, "event", "H30")
	port := r.build.Inbound.MixedPort
	if port == 0 {
		port = r.build.MixedPort
	}
	exit := routing.ExitProbe{ProxyPort: port, Store: r.Store}
	if res := exit.ProbeAndStore(ctx, profile.ID); res != nil {
		r.Log.Info("exit probe", "exit_ru", *res, "event", "H27")
	}
	r.maintenance.ScheduleAfterConnect(r.daemonCtx, profile.ID, 0, probe)
	return nil
}

func (r *Runtime) loadBulkPoolLegs(ctx context.Context, primaryID int64) []store.Profile {
	if r.Store == nil {
		return nil
	}
	ids := r.Store.GetMultipathChannelPool(ctx)
	if len(ids) == 0 {
		return nil
	}
	urlAlive := r.Store.GetBulkURLAlivePool(ctx)
	urlAliveSet := map[int64]struct{}{}
	for _, id := range urlAlive {
		urlAliveSet[id] = struct{}{}
	}
	set, _ := r.Store.LoadSettings(ctx)
	maxLegs := r.Store.EffectiveBulkMaxLegs(ctx)
	if set.BulkMaxLegs > 0 {
		maxLegs = set.BulkMaxLegs
	}
	var out []store.Profile
	seen := map[int64]struct{}{primaryID: {}}
	skippedNotURLAlive := 0
	for _, id := range ids {
		if id == primaryID {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if len(urlAliveSet) > 0 {
			if _, ok := urlAliveSet[id]; !ok {
				skippedNotURLAlive++
				continue
			}
		}
		p, err := r.Store.ProfileByID(ctx, id)
		if err != nil {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, p)
		if len(out) >= maxLegs-1 {
			break
		}
	}
	if skippedNotURLAlive > 0 && r.Log != nil {
		r.Log.Info("bulk pool filtered", "skipped_not_url_alive", skippedNotURLAlive, "url_alive", len(urlAliveSet), "event", "BULK-pool-filter")
	}
	summary := make([]map[string]any, 0, len(out))
	for _, p := range out {
		summary = append(summary, map[string]any{
			"id":   p.ID,
			"name": p.Name,
			"type": p.Type,
		})
	}
	// #region agent log
	agentDebugLog("H1,H2", "internal/controller/runtime.go:loadBulkPoolLegs", "bulk pool legs loaded from multipath channel pool", map[string]any{
		"primary_id":    primaryID,
		"max_legs":      maxLegs,
		"pool_ids":      ids,
		"url_alive_ids": urlAlive,
		"legs":          summary,
	})
	// #endregion
	return out
}

// WriteStatusFile writes desktop-control-status.txt for ctl compatibility.
func (r *Runtime) WriteStatusFile() error {
	st := r.Status()
	content := fmt.Sprintf("timestamp=%d\nstate=%s\nconnected=%t\nprofileName=%s\nselectedProxy=%s\ncurrentProfile=%d\ncurrentProfileName=%s\nserviceMode=simple\n",
		time.Now().UnixMilli(), st.State, st.ConnectedBool(), st.ProfileName, st.ProxyName, st.ProfileID, st.ProfileName)
	return os.WriteFile(r.Paths.ControlStatusFile(), []byte(content), 0o644)
}

// ExportSimpleLog writes desktop-control-export.txt path.
func (r *Runtime) ExportSimpleLog() (string, error) {
	_ = os.MkdirAll(r.Paths.CacheDir, 0o700)
	sep := string(os.PathSeparator)
	path := r.Paths.CacheDir + sep + "simple-mode.log"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		st := r.Status()
		if st.ProfileName == "" {
			st.ProfileName = "-"
		}
		_, _ = fmt.Fprintf(f, "%d state=%s connected=%t profile=%s\n",
			time.Now().UnixMilli(), st.State, st.ConnectedBool(), st.ProfileName)
		_ = f.Close()
	}
	export := r.Paths.CacheDir + sep + "desktop-control-export.txt"
	activityLog := r.Paths.CacheDir + sep + "activity.log"
	daemonLog := r.Paths.CacheDir + sep + "daemon-debug.err.log"
	body := fmt.Sprintf("timestamp=%d\npath=%s\nactivity_log=%s\ndaemon_log=%s\n",
		time.Now().UnixMilli(), path, activityLog, daemonLog)
	_ = os.WriteFile(export, []byte(body), 0o644)
	return path, nil
}
