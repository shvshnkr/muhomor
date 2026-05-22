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
	"github.com/muhomor/muhomor/internal/probe"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/reachability"
	"github.com/muhomor/muhomor/internal/routing"
	"github.com/muhomor/muhomor/internal/selector"
	"github.com/muhomor/muhomor/internal/simplemode"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

// Runtime coordinates store, mihomo, and simple mode (Phase 2).
type Runtime struct {
	Paths   paths.Layout
	Store   *store.Store
	Log     *slog.Logger
	mu      sync.Mutex
	status  Status
	mihomo  *mihomo.Client
	build   configgen.BuildOptions
	proxy   string

	connect     *simplemode.Connector
	selector    *selector.Selector
	adaptor     *simplemode.Adaptor
	health      *simplemode.SessionHealth
	maintenance *simplemode.Maintenance
	netmon      *simplemode.NetworkMonitor
	reachCache  simplemode.ReachabilityCache
	daemonCtx   context.Context
	Events      *EventHub
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
	sel.Ephemeral = &selector.EphemeralTester{MihomoBin: mihomo.ResolveBin(), Store: st}
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
		Store: r.Store,
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

func (r *Runtime) Start(ctx context.Context) error {
	r.setStatus(Status{State: StateConnecting})
	r.setActivity(ctx, "Подключение…")
	err := r.connect.Connect(ctx)
	if err != nil {
		r.setActivity(ctx, err.Error())
		r.setStatus(Status{State: StateIdle})
		return err
	}
	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	r.selector.CancelConnect()
	if r.adaptor != nil {
		r.adaptor.CancelAll()
	}
	r.reachCache.Invalidate()
	r.health.Stop()
	r.clearActivity(ctx)
	r.setStatus(Status{State: StateStopping})
	if r.mihomo != nil {
		r.mihomo.Stop()
		r.mihomo = nil
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
			r.selector.RecordFailure(profile.ID)
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
	qp, _ := r.Store.RouteQuickProfile(ctx)
	rules := routing.QuickProfileRuleLines(qp)
	rules = routing.ApplyRuGeoRules(ctx, r.Store, profile.ID, rules)
	rulesDir := r.Paths.MihomoDir() + string(os.PathSeparator) + "ruleset"
	yaml, proxyName, err := configgen.BuildFromProfileExtQP(profile, r.build, rules, rulesDir, qp)
	if err != nil {
		return err
	}
	cfgPath := r.Paths.ConfigPath()
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		return err
	}
	client, err := r.startMihomoClient(ctx, cfgPath)
	if err != nil {
		return err
	}
	r.proxy = proxyName

	r.setActivity(ctx, "Проверка соединения…")
	time.Sleep(400 * time.Millisecond)
	testURL := r.Store.ConnectionTestURL(ctx)
	testMs := r.Store.ConnectionTestTimeoutMs(ctx)
	delay, delayErr := client.ProxyDelay(ctx, proxyName, testURL, testMs)
	if delayErr != nil || delay <= 0 {
		r.mu.Lock()
		if r.mihomo != nil {
			r.mihomo.Stop()
			r.mihomo = nil
		}
		r.mu.Unlock()
		mihomo.KillAll()
		r.Log.Info("post-connect url test failed", "profile", profile.ID, "proxy", proxyName, "delay", delay, "err", delayErr, "event", "H3")
		if delayErr != nil {
			return fmt.Errorf("post-connect url test failed: %w", delayErr)
		}
		return fmt.Errorf("post-connect url test failed: no response via proxy")
	}
	r.Log.Info("post-connect url test ok", "profile", profile.ID, "delay_ms", delay, "event", "H3")

	r.health.Delay = client
	r.health.Start(r.daemonCtx, profile.ID, proxyName)

	_ = r.Store.SetCurrentProfileID(ctx, profile.ID)
	r.clearActivity(ctx)
	r.setStatus(Status{
		State:       StateConnected,
		Connected:   true,
		ProfileID:   profile.ID,
		ProfileName: profile.Name,
		ProxyName:   proxyName,
	})
	r.Log.Info("service connected", "profile", profile.Name, "proxy", proxyName, "quick_profile", qp, "event", "H30")
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

// WriteStatusFile writes desktop-control-status.txt for ctl compatibility.
func (r *Runtime) WriteStatusFile() error {
	st := r.Status()
	content := fmt.Sprintf("timestamp=%d\nstate=%s\nconnected=%t\nprofileName=%s\nselectedProxy=%s\ncurrentProfile=%d\ncurrentProfileName=%s\nserviceMode=simple\n",
		time.Now().UnixMilli(), st.State, st.ConnectedBool(), st.ProfileName, st.ProxyName, st.ProfileID, st.ProfileName)
	return os.WriteFile(r.Paths.ControlStatusFile(), []byte(content), 0o644)
}

// ExportSimpleLog writes desktop-control-export.txt path.
func (r *Runtime) ExportSimpleLog() (string, error) {
	path := r.Paths.CacheDir + string(os.PathSeparator) + "simple-mode.log"
	_ = os.MkdirAll(r.Paths.CacheDir, 0o700)
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
	export := r.Paths.CacheDir + string(os.PathSeparator) + "desktop-control-export.txt"
	body := fmt.Sprintf("timestamp=%d\npath=%s\n", time.Now().UnixMilli(), path)
	_ = os.WriteFile(export, []byte(body), 0o644)
	return path, nil
}
