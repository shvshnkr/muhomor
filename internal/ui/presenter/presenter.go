package presenter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/platform"
	"github.com/muhomor/muhomor/internal/ui/model"
)

// Presenter connects appcore to UI (Fyne / future Compose).
type Presenter struct {
	App  *appcore.App
	OnUI func(model.ConnectionUI, model.SettingsUI)
	mu            sync.Mutex
	conn          model.ConnectionUI
	set           model.SettingsUI
	connectCancel context.CancelFunc
	connecting    bool
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
	if err := p.refresh(ctx); err != nil {
		return err
	}
	go p.watchEvents(ctx)
	return nil
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
				_ = p.refresh(ctx)
			}
		}
	}
	ch, err := p.App.Events.Subscribe(ctx)
	if err != nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if ev.Status != nil {
				p.mu.Lock()
				p.applyStatus(*ev.Status)
				if p.conn.Busy && ev.Status.ActivityText != "" {
					p.conn.ActivityText = ev.Status.ActivityText
				}
				p.mu.Unlock()
				p.emit()
			}
		}
	}
}

func (p *Presenter) refresh(ctx context.Context) error {
	st, err := p.App.Service.Status(ctx)
	if err != nil {
		p.mu.Lock()
		p.conn.ErrorText = err.Error()
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
	}
	p.conn.ErrorText = ""
	p.mu.Unlock()
	p.emit()
	return nil
}

func (p *Presenter) applyStatus(st apiclient.ServiceStatus) {
	state := st.State
	connected := st.IsConnected()
	// Daemon may still report Connecting while UI/presenter already idle — avoid «Отключено» + «Подключение».
	if !connected && (state == apiclient.StateConnecting || state == apiclient.StateConnected) {
		state = apiclient.StateIdle
	}
	p.conn.State = state
	p.conn.Connected = connected
	p.conn.ProfileID = st.ProfileID
	p.conn.ProfileName = st.ProfileName
	p.conn.ProxyName = st.ProxyName
	if !p.conn.Busy {
		p.conn.ActivityText = st.ActivityText
	}
	p.conn.ProbeText = formatProbeProgress(st.Probe)
	p.conn.MultipathText = formatMultipathProgress(st.Multipath)
	p.conn.LastPingMs = st.LastPingMs
	p.conn.LastPingError = st.LastPingError
	p.conn.BulkMembers = bulkMembersFromAPI(st.BulkMembers)
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

func formatProbeProgress(pr *apiclient.ProbeProgress) string {
	if pr == nil || pr.TotalEnabled == 0 {
		return ""
	}
	line := fmt.Sprintf("Probe: alive %d / %d", pr.Alive+pr.Candidate, pr.TotalEnabled)
	if pr.SchedulerEnabled && pr.LastTickChecked > 0 {
		line += fmt.Sprintf(" · tick +%d/%d", pr.LastTickOK, pr.LastTickChecked)
	}
	if pr.LastSelectReason != "" {
		line += " · " + pr.LastSelectReason
	}
	return line
}

func (p *Presenter) emit() {
	if p.OnUI == nil {
		return
	}
	p.mu.Lock()
	c, s := p.conn, p.set
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

	p.setBusy(true, "Подключение…")
	pollCtx, stopPoll := context.WithCancel(ctx)
	defer stopPoll()
	go p.pollWhileBusy(pollCtx)
	defer p.setBusy(false, "")
	if err := p.App.SimpleConnect(ctx); err != nil {
		if ctx.Err() != nil {
			p.conn.ErrorText = model.FriendlyConnectError(ctx.Err())
			p.emit()
			_ = p.refresh(context.Background())
			return ctx.Err()
		}
		p.conn.ErrorText = model.FriendlyConnectError(err)
		p.emit()
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
	tick := time.NewTicker(400 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			_ = p.refresh(ctx)
		}
	}
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
		p.conn.ErrorText = ""
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
