package controller

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
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
	"github.com/muhomor/muhomor/internal/standby"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
	"github.com/muhomor/muhomor/internal/ui/model"
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

	connect          *simplemode.Connector
	selector         *selector.Selector
	adaptor          *simplemode.Adaptor
	health           *simplemode.SessionHealth
	maintenance      *simplemode.Maintenance
	netmon           *simplemode.NetworkMonitor
	reachCache       simplemode.ReachabilityCache
	daemonCtx        context.Context
	Events           *EventHub
	connectMu        sync.Mutex
	connectCancel    context.CancelFunc
	picker           *mihomo.Picker
	standbyRefresher *standby.Refresher
	dialWatchCancel  context.CancelFunc

	trafficMu        sync.Mutex
	trafficAt        time.Time
	trafficUp        int64
	trafficDown      int64
	trafficUpTotal   int64
	trafficDownTotal int64
	trafficStop      chan struct{}
	trafficErrs      int
	trafficErrAt     time.Time
	mihomoReloadAt   time.Time

	pingMu   sync.Mutex
	pingBusy bool
	pingStop chan struct{}

	verifier  *ConnectionVerifier
	lifecycle *LifecycleSupervisor

	restartMu        sync.Mutex
	restartWindowAt  time.Time
	restartAttempts  int
	degradedCycles   int
	pendingConnectAfterStop bool
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
		verifier:    NewConnectionVerifier(),
		lifecycle:   NewLifecycleSupervisor(),
	}
	r.picker = mihomo.NewPicker(layout, mihomo.ResolveBin())
	r.picker.Log = log
	sel.Ephemeral = &selector.EphemeralTester{MihomoBin: mihomo.ResolveBin(), Store: st, Log: log, Activity: r.setActivity, Picker: r.picker}
	r.standbyRefresher = &standby.Refresher{
		Store:      st,
		Picker:     r.picker,
		BLPrimary:  &selector.PickerSuiteRunner{Picker: r.picker},
		BLFallback: &selector.EphemeralSuiteRunner{MihomoBin: mihomo.ResolveBin()},
		Log:        log,
		Activity:   r.setActivity,
		WLOnly: func(ctx context.Context) bool {
			v, _ := st.GetKV(ctx, store.KeySimpleModeUseWLPoolOnly)
			return v == "true"
		},
		IsConnected: func(ctx context.Context) bool {
			return r.Status().State == StateConnected
		},
		ProdDelay: func(ctx context.Context, proxyName string) (int, error) {
			r.mu.Lock()
			client := r.mihomo
			r.mu.Unlock()
			if client == nil || proxyName == "" {
				return 0, fmt.Errorf("not connected")
			}
			return client.TestProxyDelay(ctx, proxyName)
		},
		ActiveSession: func(ctx context.Context) (profileID int64, proxyName string, ok bool) {
			st := r.Status()
			if st.State != StateConnected || st.ProfileID <= 0 {
				return 0, "", false
			}
			return st.ProfileID, st.ProxyName, true
		},
	}
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
			if id <= 0 {
				return fmt.Errorf("health check: no active profile")
			}
			probe, _ := r.cachedProbe(ctx)
			r.setActivity(ctx, "Проверка сессии не прошла, переключение…")
			for _, e := range standby.HotAlternates(ctx, st, id) {
				p, err := st.ProfileByID(ctx, e.ProfileID)
				if err != nil {
					continue
				}
				r.setActivity(ctx, "Переключение на hot standby…")
				if err := r.startProfile(ctx, p, probe); err == nil {
					if r.Log != nil {
						r.Log.Info("session recover", "recover_path", "hot", "profile", p.ID)
					}
					return nil
				}
			}
			if r.standbyRefresher != nil {
				r.setActivity(ctx, "Срочное обновление запасных…")
				r.standbyRefresher.RefreshUrgent(ctx, standby.CandidateBatchCap)
				for _, e := range standby.HotAlternates(ctx, st, id) {
					p, err := st.ProfileByID(ctx, e.ProfileID)
					if err != nil {
						continue
					}
					r.setActivity(ctx, "Переключение на urgent standby…")
					if err := r.startProfile(ctx, p, probe); err == nil {
						if r.Log != nil {
							r.Log.Info("session recover", "recover_path", "urgent", "profile", p.ID)
						}
						return nil
					}
				}
			}
			r.setActivity(ctx, "Восстановление сессии (полный подбор)…")
			wl := probe.WhitelistOnly()
			best, res, err := sel.Prepare(ctx, selector.PrepareOpts{
				Owner:          selector.OwnerSessionRecover,
				NetworkHandoff: true,
				WhitelistOnly:  wl,
			})
			if err == nil && res == selector.ResultSuccess {
				if r.Log != nil {
					r.Log.Info("session recover", "recover_path", "cold", "profile", best.ID)
				}
				return r.startProfile(ctx, best, probe)
			}
			next, ok := sel.TryMoveToFallback(ctx, id)
			if !ok {
				sel.RecordFailure(id)
				return fmt.Errorf("no fallback")
			}
			if r.Log != nil {
				r.Log.Info("session recover", "recover_path", "fallback", "profile", next.ID)
			}
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
	r.build.ExternalController = paths.DefaultExternalController()
	if set.ServiceMode == store.ServiceModeVPN {
		r.build.Tun.Enable = true
	}
}

func (r *Runtime) SetDaemonContext(ctx context.Context) {
	r.daemonCtx = ctx
	r.netmon.Start(ctx)
	rulesDir := r.Paths.MihomoDir() + string(os.PathSeparator) + "ruleset"
	cfgDir := r.Paths.MihomoDir()
	sched := &Scheduler{
		Store:   r.Store,
		Assets:  &subscription.AssetUpdater{RulesDir: rulesDir},
		Subs:    r.connect.Updater,
		Log:     r.Log,
		GeoDir:  cfgDir,
		GeoBusy: func() bool { return r.connectInFlight() },
	}
	go sched.Run(ctx)
	if r.picker != nil {
		go r.runPickerEnsureWithRetry(ctx)
	}
	go func() {
		if mihomo.GeoDatabaseCached(cfgDir) {
			return
		}
		bg, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
		defer cancel()
		if err := mihomo.DownloadGeoDatabase(bg, cfgDir); err != nil && r.Log != nil {
			r.Log.Warn("geo prefetch", "err", err)
		}
	}()
	if st := r.Store; st != nil && st.ProbeSchedulerEnabled(ctx) {
		ps := &probe.Scheduler{Store: st, Log: r.Log, Config: probe.ConfigFromStore(ctx, st)}
		go ps.Run(ctx)
	}
	if r.standbyRefresher != nil {
		go r.standbyRefresher.Run(ctx)
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
	if _, err := r.startMihomoClient(ctx, cfgPath, false); err != nil {
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
	_ = r.WriteStatusFile()
}

func (r *Runtime) patchStatus(s Status) {
	r.mu.Lock()
	r.status = s
	r.mu.Unlock()
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
		d, err := r.pingWithBulkFallback(ctx, client, proxy)
		out.DelayMs = d
		if err != nil {
			out.Error = model.FriendlyDelayError(err.Error())
		}
		_ = r.Store.SetKV(ctx, store.KeyLastServicePingMs, fmt.Sprintf("%d", out.DelayMs))
		if out.Error != "" {
			_ = r.Store.SetKV(ctx, store.KeyLastServicePingError, out.Error)
		} else {
			_ = r.Store.SetKV(ctx, store.KeyLastServicePingError, "")
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

func (r *Runtime) connectInFlight() bool {
	r.connectMu.Lock()
	defer r.connectMu.Unlock()
	return r.connectCancel != nil
}

func (r *Runtime) Start(_ context.Context) error {
	if r.isStopping() {
		r.queueConnectAfterStop()
		if r.Log != nil {
			r.Log.Info("connect deferred while stopping", "connect_block_reason", "stopping", "event", "H4-connect-queued")
		}
		r.setActivity(context.Background(), "Остановка в процессе, подключение будет запущено сразу после завершения…")
		return nil
	}
	// Long connect must not use HTTP request ctx (client disconnect / GUI cancel would abort probes).
	ctx := r.beginConnect(context.Background())
	defer r.endConnect()
	r.mu.Lock()
	alreadyUp := r.mihomo != nil && r.proxy != ""
	r.mu.Unlock()
	if alreadyUp {
		r.setActivity(ctx, "Переподключение…")
	} else {
		r.setStatus(Status{State: StateConnecting})
		r.setActivity(ctx, "Подключение…")
	}
	err := r.connect.Connect(ctx)
	if err != nil {
		if ctx.Err() != nil {
			r.clearActivity(context.Background())
			r.setStatus(Status{State: StateStopped})
			return fmt.Errorf("connect aborted")
		}
		r.setActivity(ctx, err.Error())
		if r.verifier != nil {
			if strings.Contains(err.Error(), "all probes dead") || strings.Contains(err.Error(), "живых") {
				r.verifier.MarkNoLiveServers(err.Error())
			}
		}
		r.setStatus(Status{State: StateIdle})
		return err
	}
	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	if r.lifecycle == nil {
		r.lifecycle = NewLifecycleSupervisor()
	}
	var launchPending bool
	finalize := func() {
		r.clearActivity(context.Background())
		r.setStatus(Status{State: StateStopped})
		launchPending = r.consumeQueuedConnectAfterStop()
	}
	defer func() {
		finalize()
		if launchPending {
			go func() {
				if err := r.Start(context.Background()); err != nil && r.Log != nil {
					r.Log.Warn("queued connect after stop failed", "err", err, "event", "H4-connect-queued")
				}
			}()
		}
	}()

	r.cancelInFlightConnect()
	r.selector.CancelConnect()
	if r.adaptor != nil {
		r.adaptor.CancelAll()
	}
	r.reachCache.Invalidate()
	r.health.Stop()
	r.stopDialWatchdog()
	r.clearActivity(ctx)
	r.setStatus(Status{State: StateStopping})
	r.stopCurrentMihomo()
	if r.picker != nil {
		r.picker.Stop()
	}
	r.stopTrafficSampler()
	r.stopPingSampler()
	r.clearTrafficCache()
	r.clearLastPing(ctx)
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
	if r.lifecycle == nil {
		r.lifecycle = NewLifecycleSupervisor()
	}
	err := r.lifecycle.Do(LifecycleReloading, func() error {
		if proxy != "" {
			if d, err := client.TestProxyDelay(ctx, proxy); err == nil && d > 0 {
				return client.Reload(ctx)
			}
		}
		return r.ReapplyCurrentProfile(ctx)
	})
	if err == nil {
		r.markMihomoReload()
	}
	return err
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

func (r *Runtime) postConnectSwitchBudget(ctx context.Context) int {
	const max = 8
	if r.Store == nil {
		return max
	}
	q, err := r.Store.FallbackQueue(ctx)
	if err != nil || len(q) == 0 {
		return max
	}
	if len(q) < max {
		return len(q)
	}
	return max
}

func (r *Runtime) startProfile(ctx context.Context, profile store.Profile, probe reachability.Result) error {
	if profile.ID <= 0 {
		return fmt.Errorf("invalid profile id %d", profile.ID)
	}
	maxPostConnectSwitch := r.postConnectSwitchBudget(ctx)
	for attempt := 0; attempt < maxPostConnectSwitch; attempt++ {
		if attempt > 0 {
			r.recordRestartAttempt()
			r.setActivity(ctx, fmt.Sprintf("Перезапуск прокси (попытка %d)…", attempt+1))
			if ev := r.selector.LastProbeEvidence(); ev.Degraded {
				r.bumpDegradedCycle()
				// Throttle restart pressure in degraded URL phase.
				backoff := time.Duration(attempt) * 700 * time.Millisecond
				if backoff > 4*time.Second {
					backoff = 4 * time.Second
				}
				time.Sleep(backoff)
			}
		}
		if err := r.startProfileAttempt(ctx, profile, probe); err != nil {
			if attempt+1 >= maxPostConnectSwitch {
				if r.verifier != nil {
					r.verifier.MarkNoLiveServers(err.Error())
				}
				r.resetAfterProfileFailure(ctx)
				return err
			}
			var next store.Profile
			var ok bool
			hotIdx := 0
			hotAlts := standby.HotAlternates(ctx, r.Store, profile.ID)
			for hotIdx < len(hotAlts) {
				p, perr := r.Store.ProfileByID(ctx, hotAlts[hotIdx].ProfileID)
				hotIdx++
				if perr != nil {
					continue
				}
				next, ok = p, true
				break
			}
			if !ok {
				next, ok = r.selector.TryMoveToFallback(ctx, profile.ID)
			}
			if !ok {
				if r.verifier != nil {
					r.verifier.MarkNoLiveServers(err.Error())
				}
				r.resetAfterProfileFailure(ctx)
				return err
			}
			r.setActivity(ctx, fmt.Sprintf("Сервер нестабилен (%d/%d), переключение…", attempt+1, maxPostConnectSwitch))
			profile = next
			continue
		}
		if q := standby.ShortFallbackQueue(ctx, r.Store, profile.ID); len(q) > 0 {
			_ = r.Store.SetFallbackQueue(ctx, q)
		}
		return nil
	}
	r.resetAfterProfileFailure(ctx)
	if r.verifier != nil {
		r.verifier.MarkNoLiveServers("post-connect: fallbacks exhausted")
	}
	return fmt.Errorf("post-connect: fallbacks exhausted")
}

func (r *Runtime) stopDialWatchdog() {
	if r.dialWatchCancel != nil {
		r.dialWatchCancel()
		r.dialWatchCancel = nil
	}
}

func (r *Runtime) startDialWatchdog(logPath string, logOffset int64) {
	r.stopDialWatchdog()
	if r.daemonCtx == nil || logPath == "" {
		return
	}
	ctx, cancel := context.WithCancel(r.daemonCtx)
	r.dialWatchCancel = cancel
	onBurst := func() {
		if r.Log != nil {
			r.Log.Info("dial timeout burst", "event", "H34-dial")
		}
		id, _ := r.Store.CurrentProfileID(ctx)
		if id <= 0 || r.health == nil || r.health.OnUnhealthy == nil {
			return
		}
		if r.standbyRefresher != nil {
			r.standbyRefresher.RefreshUrgent(ctx, standby.CandidateBatchCap)
		}
		_ = r.health.OnUnhealthy(ctx, id)
	}
	go mihomo.RunDialWatchdog(ctx, logPath, logOffset, onBurst)
}

func (r *Runtime) resetAfterProfileFailure(ctx context.Context) {
	r.stopPingSampler()
	if r.Store != nil {
		_ = r.Store.SetKV(ctx, store.KeyLastServicePingMs, "")
		_ = r.Store.SetKV(ctx, store.KeyLastServicePingError, "")
	}
	r.stopCurrentMihomo()
	r.mu.Lock()
	r.proxy = ""
	r.mu.Unlock()
	r.clearActivity(ctx)
	r.setStatus(Status{State: StateIdle})
}

func (r *Runtime) startProfileAttempt(ctx context.Context, profile store.Profile, probe reachability.Result) error {
	if profile.ID <= 0 {
		return fmt.Errorf("invalid profile id %d", profile.ID)
	}
	if r.verifier != nil {
		r.verifier.StartAttempt()
		ev := r.selector.LastProbeEvidence()
		r.verifier.ObserveSelection(ev.TCPOK, ev.URLOK, ev.PoolSize)
	}
	r.refreshBuildOptions(ctx)
	prev := r.Status()
	reapplySame := prev.State == StateConnected && prev.ProfileID == profile.ID
	if !reapplySame {
		r.stopDialWatchdog()
	}
	if reapplySame {
		r.setActivity(ctx, "Обновление конфигурации…")
	} else {
		r.setActivity(ctx, "Запуск mihomo…")
		r.setStatus(Status{State: StateConnecting, ProfileID: profile.ID, ProfileName: profile.Name})
	}
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
	cfgPath := r.Paths.ConfigPath()
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		return err
	}
	cfgDir := r.Paths.MihomoDir()
	logPath := mihomo.SubprocessLogPath(cfgDir)
	logOffset := mihomo.LogFileSize(logPath)
	if !mihomo.GeoDatabaseCached(cfgDir) {
		r.setActivity(ctx, "Запуск mihomo…")
	}
	mihomo.RunGeoStartupWatcher(ctx, cfgDir, logPath, logOffset, func(text string) {
		r.setActivity(ctx, text)
	})
	r.mu.Lock()
	tryReload := r.mihomo != nil && !reapplySame
	r.mu.Unlock()
	client, err := r.startMihomoClient(ctx, cfgPath, tryReload)
	if err != nil {
		return err
	}
	if r.verifier != nil {
		r.verifier.MarkTransportAlive("mihomo_api_ready")
	}
	effectiveProxy := proxyName
	if plan.Active && plan.MatchTarget != "" {
		effectiveProxy = plan.MatchTarget
	}
	r.proxy = effectiveProxy

	r.setActivity(ctx, fmt.Sprintf("Проверка соединения через %s…", effectiveProxy))
	_ = client.WaitProxyReady(ctx, effectiveProxy, 8*time.Second)
	testURL := r.Store.ConnectionTestURL(ctx)
	testMs := r.Store.ConnectionTestTimeoutMs(ctx)
	ev := r.selector.LastProbeEvidence()
	degradedPrepare := ev.Degraded
	if degradedPrepare {
		if testMs <= 0 || testMs > 4000 {
			testMs = 4000
		}
		r.setActivity(ctx, "Ускоренная проверка degraded-сессии…")
	} else if testMs < 8000 {
		testMs = 8000
	}
	logRetry := func(msg string, args ...any) {
		if r.Log != nil {
			r.Log.Info(msg, args...)
		}
	}
	_, delay, delayErr := postConnectDelay(ctx, logRetry, client, plan, proxyName, testURL, testMs)
	if degradedPrepare && (delayErr != nil || delay <= 0) {
		quickRetryBudget := 2
		for i := 0; i < quickRetryBudget; i++ {
			if ctx.Err() != nil {
				break
			}
			time.Sleep(350 * time.Millisecond)
			_, delay, delayErr = postConnectDelay(ctx, logRetry, client, plan, proxyName, testURL, testMs)
			if delayErr == nil && delay > 0 {
				break
			}
		}
	}
	if delayErr != nil || delay <= 0 {
		if r.Store != nil {
			_ = r.Store.RemoveFromBulkURLAlivePool(ctx, profile.ID)
		}
		r.mu.Lock()
		r.proxy = ""
		r.mu.Unlock()
		r.Log.Info("post-connect url test failed", "profile", profile.ID, "proxy", effectiveProxy, "primary_proxy", proxyName, "prepare_decision", ev.PrepareDecision, "delay", delay, "err", delayErr, "event", "H3")
		if delayErr != nil {
			if r.verifier != nil {
				r.verifier.MarkTransportAlive("delay_probe_failed")
			}
			if msg := model.FriendlyDelayError(delayErr.Error()); msg != "" {
				r.setActivity(ctx, msg)
			} else {
				r.setActivity(ctx, "Проверка соединения не прошла")
			}
			return fmt.Errorf("post-connect url test failed: %w", delayErr)
		}
		r.setActivity(ctx, "Нет ответа через прокси")
		return fmt.Errorf("post-connect url test failed: no response via proxy")
	}
	if r.verifier != nil {
		r.verifier.MarkQualityVerified("delay_probe_ok")
	}
	r.Log.Info("post-connect url test ok", "profile", profile.ID, "proxy", effectiveProxy, "prepare_decision", ev.PrepareDecision, "delay_ms", delay, "event", "H3")
	standby.PromoteAfterConnect(ctx, r.Store, profile.ID, effectiveProxy, delay)
	r.startDialWatchdog(logPath, logOffset)

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
	if r.daemonCtx != nil {
		r.startTrafficSampler(r.daemonCtx)
		r.startPingSampler(r.daemonCtx)
	}
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
	return out
}

func (r *Runtime) runPickerEnsureWithRetry(ctx context.Context) {
	if r.picker == nil {
		return
	}
	delays := []time.Duration{0, 7 * time.Second, 12 * time.Second}
	for attempt, wait := range delays {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		}
		bg, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		err := r.picker.Ensure(bg)
		cancel()
		if err == nil {
			return
		}
		if r.Log != nil {
			r.Log.Warn("picker ensure", "attempt", attempt+1, "err", err)
		}
	}
}

func (r *Runtime) stopCurrentMihomo() {
	if r.lifecycle == nil {
		r.lifecycle = NewLifecycleSupervisor()
	}
	_ = r.lifecycle.Do(LifecycleStopping, func() error {
		r.mu.Lock()
		client := r.mihomo
		r.mihomo = nil
		r.mu.Unlock()
		if client != nil {
			client.Stop()
		}
		return nil
	})
}

func (r *Runtime) recordRestartAttempt() {
	r.restartMu.Lock()
	defer r.restartMu.Unlock()
	now := time.Now()
	if r.restartWindowAt.IsZero() || now.Sub(r.restartWindowAt) > 5*time.Minute {
		r.restartWindowAt = now
		r.restartAttempts = 0
	}
	r.restartAttempts++
	if r.Log != nil {
		r.Log.Info("restart attempt", "count_window", r.restartAttempts, "window_sec", 300, "event", "H3-restart-window")
	}
}

func (r *Runtime) bumpDegradedCycle() {
	r.restartMu.Lock()
	r.degradedCycles++
	n := r.degradedCycles
	r.restartMu.Unlock()
	if r.Log != nil {
		r.Log.Info("degraded cycle", "cycles", n, "event", "H4-degraded-cycle")
	}
}

func (r *Runtime) isStopping() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status.State == StateStopping
}

func (r *Runtime) queueConnectAfterStop() {
	r.mu.Lock()
	r.pendingConnectAfterStop = true
	r.mu.Unlock()
}

func (r *Runtime) consumeQueuedConnectAfterStop() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.pendingConnectAfterStop {
		return false
	}
	r.pendingConnectAfterStop = false
	return true
}

// WriteStatusFile writes desktop-control-status.txt for ctl compatibility.
func (r *Runtime) WriteStatusFile() error {
	st := r.statusSnapshot(context.Background())
	content := fmt.Sprintf("timestamp=%d\nstate=%s\nconnected=%t\nconnected_verified=%t\nconnected_degraded=%t\nlive_servers_confirmed=%t\nverification_phase=%s\nprofileName=%s\nselectedProxy=%s\ncurrentProfile=%d\ncurrentProfileName=%s\nserviceMode=simple\n",
		time.Now().UnixMilli(), st.State, st.Connected, st.ConnectedVerified, st.ConnectedDegraded, st.LiveServersConfirmed, st.VerificationPhase, st.ProfileName, st.ProxyName, st.ProfileID, st.ProfileName)
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
	sts := r.statusSnapshot(context.Background())
	body := fmt.Sprintf("timestamp=%d\npath=%s\nactivity_log=%s\ndaemon_log=%s\nverification_phase=%s\nlive_servers_confirmed=%t\n",
		time.Now().UnixMilli(), path, activityLog, daemonLog, sts.VerificationPhase, sts.LiveServersConfirmed)
	_ = os.WriteFile(export, []byte(body), 0o644)
	return path, nil
}
