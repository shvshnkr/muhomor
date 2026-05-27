package presenter

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/platform"
	"github.com/muhomor/muhomor/internal/ui/model"
)

// Presenter connects appcore to UI (Fyne / future Compose).
type Presenter struct {
	App           *appcore.App
	OnUI          func(model.ConnectionUI, model.SettingsUI)
	mu            sync.Mutex
	conn          model.ConnectionUI
	set           model.SettingsUI
	connectCancel context.CancelFunc
	connecting    bool
	refreshBusy   bool
	shuttingDown  bool
	pinnedError   string
	sseCancel     context.CancelFunc
	watchStarted  sync.Once
}

// New builds presenter with update callback.
func New(app *appcore.App, onUI func(model.ConnectionUI, model.SettingsUI)) *Presenter {
	return &Presenter{App: app, OnUI: onUI}
}

// Start ensures daemon and begins SSE subscription.
func (p *Presenter) Start(ctx context.Context, daemonArgs []string) error {
	if err := platform.EnsureDaemon(ctx, p.App.Layout, daemonArgs); err != nil {
		return err
	}
	var last error
	for attempt := 0; attempt < 5; attempt++ {
		if err := p.refresh(ctx); err == nil {
			last = nil
			break
		} else {
			last = err
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(400 * time.Millisecond):
			}
		}
	}
	if last != nil {
		return last
	}
	p.watchStarted.Do(func() { go p.watchEvents(ctx) })
	return nil
}

func (p *Presenter) stopSSE() {
	p.mu.Lock()
	cancel := p.sseCancel
	p.sseCancel = nil
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (p *Presenter) watchEvents(ctx context.Context) {
	if p.App.Events == nil {
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				p.refreshIfIdle(ctx)
			}
		}
	}
	for {
		if ctx.Err() != nil {
			p.stopSSE()
			return
		}
		p.stopSSE()
		subCtx, cancel := context.WithCancel(ctx)
		p.mu.Lock()
		p.sseCancel = cancel
		p.mu.Unlock()

		ch, err := p.App.Events.Subscribe(subCtx)
		if err != nil {
			if p.shuttingDown {
				return
			}
			p.setDaemonUnreachable(err)
			select {
			case <-ctx.Done():
				p.stopSSE()
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}
		for {
			select {
			case <-ctx.Done():
				p.stopSSE()
				return
			case ev, ok := <-ch:
				if !ok {
					goto reconnect
				}
				if ev.Status != nil {
					p.mu.Lock()
					if p.shuttingDown {
						p.mu.Unlock()
						continue
					}
					p.conn.ErrorText = ""
					p.applyStatus(*ev.Status)
					if p.conn.Busy && ev.Status.ActivityText != "" {
						p.conn.ActivityText = ev.Status.ActivityText
					}
					p.mu.Unlock()
					p.emit()
				}
			}
		}
	reconnect:
		p.stopSSE()
		p.mu.Lock()
		sd := p.shuttingDown
		p.mu.Unlock()
		if sd {
			return
		}
		_ = p.refresh(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

// SetShuttingDown suppresses daemon/SSE errors while the GUI is exiting.
func (p *Presenter) SetShuttingDown(v bool) {
	p.mu.Lock()
	p.shuttingDown = v
	if v {
		p.conn.ErrorText = ""
		p.pinnedError = ""
		p.conn.Busy = false
		p.conn.ActivityText = ""
		p.connecting = false
	}
	p.mu.Unlock()
	if v {
		p.stopSSE()
		p.emit()
	}
}

func (p *Presenter) refresh(ctx context.Context) error {
	p.mu.Lock()
	if p.shuttingDown {
		p.mu.Unlock()
		return ctx.Err()
	}
	p.mu.Unlock()

	st, err := p.App.Service.Status(ctx)
	if err != nil {
		p.mu.Lock()
		if p.shuttingDown {
			p.mu.Unlock()
			return err
		}
		if !p.shouldSuppressDaemonBlip() {
			if err != nil {
				p.noteError(err.Error())
			}
			if isDaemonUnreachable(err) && !p.conn.Connected {
				p.clearConnectionLocked()
			}
		}
		p.mu.Unlock()
		p.emit()
		return err
	}
	set, _ := p.App.Config.LoadSettings(ctx)
	p.mu.Lock()
	p.applyStatus(st)
	p.set = model.SettingsUI{
		ServiceMode:              set.ServiceMode,
		MixedPort:                set.MixedPort,
		RouteQuick:               set.RouteQuickProfile,
		MultipathEnabled:         set.MultipathEnabled,
		MultipathPreset:          set.MultipathPreset,
		MultipathWLEmergencyOnly: set.MultipathWLEmergencyOnly,
		WLBuiltinConnectEnabled:  set.WLBuiltinConnectEnabled,
		UIKeepErrorsOnScreen:     set.UIKeepErrorsOnScreen,
	}
	p.clearLiveErrorLocked()
	p.restorePinnedErrorLocked()
	p.mu.Unlock()
	p.emit()
	return nil
}

func (p *Presenter) keepErrorsOnScreen() bool {
	return p.set.UIKeepErrorsOnScreen
}

func (p *Presenter) clearLiveErrorLocked() {
	p.conn.ErrorText = ""
	if !p.keepErrorsOnScreen() {
		p.pinnedError = ""
	}
}

func (p *Presenter) restorePinnedErrorLocked() {
	if p.keepErrorsOnScreen() && p.pinnedError != "" {
		p.conn.ErrorText = p.pinnedError
	}
}

func (p *Presenter) noteError(msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}
	p.conn.ErrorText = msg
	if p.keepErrorsOnScreen() {
		p.pinnedError = msg
	}
}

// ClearPinnedError dismisses a pinned error (when «оставлять ошибки на экране» is on).
func (p *Presenter) ClearPinnedError() {
	p.mu.Lock()
	p.pinnedError = ""
	p.conn.ErrorText = ""
	p.mu.Unlock()
	p.emit()
}

func (p *Presenter) applyStatus(st apiclient.ServiceStatus) {
	if !p.keepErrorsOnScreen() || p.pinnedError == "" {
		p.conn.ErrorText = ""
	}
	state := st.State
	connected := st.Connected || st.IsConnected()
	p.conn.State = state
	p.conn.Connected = connected
	p.conn.ConnectedVerified = st.ConnectedVerified
	p.conn.ConnectedDegraded = st.ConnectedDegraded
	p.conn.LiveServersConfirmed = st.LiveServersConfirmed
	p.conn.VerificationPhase = string(st.VerificationPhase)
	p.conn.VerificationReason = st.VerificationReason
	p.conn.ProfileID = st.ProfileID
	p.conn.ProfileName = st.ProfileName
	p.conn.ProxyName = st.ProxyName
	act := st.ActivityText
	connecting := p.conn.Busy || state == apiclient.StateConnecting
	if !connecting {
		if connected && isStaleStartupActivity(act) {
			act = ""
		}
		if !connected && isStaleStartupActivity(act) {
			act = ""
		}
	}
	title := model.ConnectStatusTitle(connected, connecting && !connected, "")
	act = model.FilterActivityForDisplay(title, act)
	if !p.conn.Busy || (act != "" && !strings.HasPrefix(strings.TrimSpace(act), "Отмена")) {
		p.conn.ActivityText = act
	}
	p.conn.ProbeText = formatProbeProgress(st.Probe, st)
	p.conn.StandbyText = formatStandbyProgress(st.Standby)
	p.conn.MultipathText = formatMultipathProgress(st.Multipath)
	if connected {
		p.conn.LastPingMs = st.LastPingMs
		if p.conn.Busy || state == apiclient.StateConnecting {
			p.conn.LastPingError = ""
		} else {
			p.conn.LastPingError = st.LastPingError
		}
		p.conn.TrafficUp = st.TrafficUp
		p.conn.TrafficDown = st.TrafficDown
	} else {
		p.conn.LastPingMs = 0
		p.conn.LastPingError = ""
		p.conn.TrafficUp = 0
		p.conn.TrafficDown = 0
	}
	p.conn.BulkMembers = bulkMembersFromAPI(st.BulkMembers)
	if connected {
		p.pinnedError = ""
		p.conn.ErrorText = ""
	} else {
		p.restorePinnedErrorLocked()
	}
}

func bulkMembersFromAPI(in []apiclient.BulkMemberStatus) []model.BulkMemberUI {
	if len(in) == 0 {
		return nil
	}
	out := make([]model.BulkMemberUI, len(in))
	for i, m := range in {
		name := m.Name
		if name == "" {
			name = m.Tag
		}
		out[i] = model.BulkMemberUI{Tag: m.Tag, Name: name, DelayMs: m.DelayMs, Error: m.Error}
	}
	return out
}

func formatMultipathProgress(mp *apiclient.MultipathProgress) string {
	if mp == nil {
		return "Пул: один туннель (Настройки)"
	}
	if mp.AggregationMode == "flow_aggregate" {
		if mp.BulkActive {
			return fmt.Sprintf("Пул load-balance: %d туннелей (мин. %d)", mp.BulkMemberCount, mp.BulkMinLegs)
		}
		reason := bulkFallbackReasonRU(mp.BulkFallbackReason)
		if reason == "" {
			reason = "ожидание подключения"
		}
		return "Пул load-balance: " + reason
	}
	if !mp.Enabled {
		return "Multipath: выкл (классический отбор)"
	}
	line := fmt.Sprintf("Multipath: %d каналов (%d healthy)", mp.ActiveChannels, mp.HealthyChannels)
	if mp.Preset != "" {
		line += ", preset=" + mp.Preset
	}
	if mp.LastReason != "" {
		line += " · " + mp.LastReason
	} else if mp.ActiveChannels == 0 {
		line += " · пул после connect"
	}
	return line
}

func bulkFallbackReasonRU(reason string) string {
	switch reason {
	case "disabled":
		return "выключен в настройках"
	case "not_enough_legs":
		return "мало туннелей"
	case "legacy_mode":
		return "режим «один туннель»"
	case "config_error":
		return "ошибка конфига"
	default:
		return reason
	}
}

func isStaleStartupActivity(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	return strings.HasPrefix(text, "Запуск mihomo") ||
		strings.HasPrefix(text, "Подключение") ||
		strings.HasPrefix(text, "Переподключение") ||
		strings.HasPrefix(text, "Обновление конфигурации") ||
		strings.HasPrefix(text, "Проверка соединения") ||
		strings.HasPrefix(text, "Сервер нестабилен")
}

func formatStandbyProgress(sp *apiclient.StandbyProgress) string {
	if sp == nil {
		return ""
	}
	hotN := len(sp.Hot)
	warmN := len(sp.Warm)
	if hotN == 0 && warmN == 0 {
		if sp.PickerRunning {
			return "Standby: пусто (picker)"
		}
		return ""
	}
	line := fmt.Sprintf("Standby: hot %d", hotN)
	if warmN > 0 {
		line += fmt.Sprintf(", warm %d", warmN)
	}
	if hotN > 0 && sp.Hot[0].DelayMs > 0 {
		line += fmt.Sprintf(" · %d ms", sp.Hot[0].DelayMs)
	}
	if !sp.PickerRunning {
		line += " · picker off"
	}
	return line
}

func formatProbeProgress(pr *apiclient.ProbeProgress, st apiclient.ServiceStatus) string {
	if st.VerificationPhase == apiclient.VerificationNoLiveServers {
		return "Живых серверов не найдено"
	}
	if pr == nil || pr.TotalEnabled == 0 {
		if st.ConnectedDegraded {
			return "Транспорт поднят, URL-проверка деградирована"
		}
		return ""
	}
	alive := pr.Alive + pr.Candidate
	line := fmt.Sprintf("Probe: alive %d / %d", alive, pr.TotalEnabled)
	if !pr.SchedulerEnabled {
		if alive == 0 {
			return "Подбор по TCP (URL-тест недоступен)"
		}
		line += " · подбор по TCP"
	} else if pr.LastTickChecked > 0 {
		line += fmt.Sprintf(" · tick +%d/%d", pr.LastTickOK, pr.LastTickChecked)
	}
	if pr.LastSelectReason != "" {
		line += " · " + pr.LastSelectReason
	}
	if st.ConnectedDegraded {
		line += " · режим degraded"
	}
	return line
}

func (p *Presenter) emit() {
	if p.OnUI == nil {
		return
	}
	p.mu.Lock()
	c, s := p.conn, p.set
	if p.shuttingDown {
		c.ErrorText = ""
		c.ActivityText = ""
		c.Busy = false
		c.LastPingError = ""
	}
	p.mu.Unlock()
	p.OnUI(c, s)
}

// Connect simple mode.
func (p *Presenter) Connect(ctx context.Context) error {
	p.mu.Lock()
	if p.connecting {
		p.mu.Unlock()
		return nil
	}
	p.connecting = true
	p.pinnedError = ""
	p.clearLiveErrorLocked()
	ctx, cancel := context.WithCancel(ctx)
	p.connectCancel = cancel
	p.mu.Unlock()
	defer func() {
		cancel()
		p.mu.Lock()
		p.connecting = false
		p.connectCancel = nil
		p.mu.Unlock()
	}()

	p.setBusy(true, "")
	pollCtx, stopPoll := context.WithCancel(ctx)
	defer stopPoll()
	go p.pollWhileBusy(pollCtx)
	defer p.setBusy(false, "")
	if err := p.App.SimpleConnect(ctx); err != nil {
		p.mu.Lock()
		if ctx.Err() != nil {
			p.noteError(model.FriendlyConnectError(ctx.Err()))
		} else {
			p.noteError(model.FriendlyConnectError(err))
		}
		p.mu.Unlock()
		p.emit()
		if ctx.Err() != nil {
			_ = p.refresh(context.Background())
			return ctx.Err()
		}
		return err
	}
	return p.refresh(ctx)
}

// AbortConnect stops daemon connect first, then aborts client wait (Отменить).
func (p *Presenter) AbortConnect(ctx context.Context) {
	_ = ctx
	p.setBusy(true, "Отмена…")
	go func() {
		_ = p.App.Disconnect(context.Background())
		p.mu.Lock()
		if p.connectCancel != nil {
			p.connectCancel()
			p.connectCancel = nil
		}
		p.connecting = false
		p.mu.Unlock()
		p.setBusy(false, "")
		_ = p.refresh(context.Background())
	}()
}

func (p *Presenter) pollWhileBusy(ctx context.Context) {
	// Activity comes from SSE; polling GET /status during POST /service/start caused false "daemon" errors.
	<-ctx.Done()
}

func (p *Presenter) refreshIfIdle(ctx context.Context) {
	p.mu.Lock()
	if p.refreshBusy || p.shuttingDown {
		p.mu.Unlock()
		return
	}
	p.refreshBusy = true
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.refreshBusy = false
		p.mu.Unlock()
	}()
	_ = p.refresh(ctx)
}

func isDaemonUnreachable(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "daemon not running")
}

func (p *Presenter) shouldSuppressDaemonBlip() bool {
	if p.shuttingDown {
		return true
	}
	if p.conn.Connected {
		return true
	}
	return p.conn.Busy || p.connecting || p.conn.State == apiclient.StateConnecting
}

func (p *Presenter) setDaemonUnreachable(err error) {
	p.mu.Lock()
	if p.shuttingDown || p.shouldSuppressDaemonBlip() {
		p.mu.Unlock()
		return
	}
	if err != nil {
		p.noteError(err.Error())
	}
	if isDaemonUnreachable(err) && !p.conn.Connected {
		p.clearConnectionLocked()
	}
	p.mu.Unlock()
	p.emit()
}

func (p *Presenter) clearConnectionLocked() {
	p.conn.Connected = false
	p.conn.State = apiclient.StateIdle
	p.conn.Busy = false
	p.conn.TrafficUp = 0
	p.conn.TrafficDown = 0
	p.conn.LastPingMs = 0
	p.conn.LastPingError = ""
}

// Disconnect stops service.
func (p *Presenter) Disconnect(ctx context.Context) error {
	p.setBusy(true, "")
	defer p.setBusy(false, "")
	if err := p.App.Disconnect(ctx); err != nil {
		p.setBusy(false, err.Error())
		return err
	}
	return p.refresh(ctx)
}

func (p *Presenter) setBusy(busy bool, activity string) {
	p.mu.Lock()
	p.conn.Busy = busy
	if activity != "" {
		p.conn.ActivityText = activity
		p.clearLiveErrorLocked()
		p.restorePinnedErrorLocked()
	}
	if !busy {
		p.conn.ActivityText = ""
	}
	p.mu.Unlock()
	p.emit()
}

// SetServiceMode updates settings and reloads.
func (p *Presenter) SetServiceMode(ctx context.Context, mode string) error {
	set, err := p.App.Config.LoadSettings(ctx)
	if err != nil {
		return err
	}
	set.ServiceMode = mode
	set.TunEnable = mode == appcore.ServiceModeVPN
	if _, err := p.App.Config.SaveSettings(ctx, set); err != nil {
		return err
	}
	st, _ := p.App.Service.Status(ctx)
	if st.IsConnected() {
		return p.App.Service.Reload(ctx)
	}
	return p.refresh(ctx)
}

// Refresh polls daemon status (for terminal pseudo-GUI).
func (p *Presenter) Refresh(ctx context.Context) error {
	return p.refresh(ctx)
}

// Ping active proxy (PROXY_BULK uses best leg fallback).
func (p *Presenter) Ping(ctx context.Context) (apiclient.PingResponse, error) {
	resp, err := p.App.Service.Ping(ctx)
	if err != nil {
		return resp, err
	}
	_ = p.refresh(ctx)
	return resp, nil
}

// BulkPingAll probes every PROXY_BULK leg in parallel.
func (p *Presenter) BulkPingAll(ctx context.Context) (apiclient.BulkPingAllResponse, error) {
	resp, err := p.App.Service.BulkPingAll(ctx)
	if err != nil {
		return resp, err
	}
	_ = p.refresh(ctx)
	return resp, nil
}

// HasBulkPool reports whether load-balance pool legs are configured.
func (p *Presenter) HasBulkPool() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.conn.BulkMembers) > 0
}

// Snapshot returns current UI state.
func (p *Presenter) Snapshot() (model.ConnectionUI, model.SettingsUI) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.conn, p.set
}

// ExportLog exports simple mode log path.
func (p *Presenter) ExportLog(ctx context.Context) (string, error) {
	res, err := p.App.Service.ExportLog(ctx)
	if err != nil {
		return "", err
	}
	if path, ok := res["path"].(string); ok {
		return path, nil
	}
	return "", fmt.Errorf("no path in response")
}
