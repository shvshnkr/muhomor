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
	mu   sync.Mutex
	conn model.ConnectionUI
	set  model.SettingsUI
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
		ServiceMode: set.ServiceMode,
		MixedPort:   set.MixedPort,
		RouteQuick:  set.RouteQuickProfile,
	}
	p.conn.ErrorText = ""
	p.mu.Unlock()
	p.emit()
	return nil
}

func (p *Presenter) applyStatus(st apiclient.ServiceStatus) {
	p.conn = model.ConnectionUI{
		State:        st.State,
		Connected:    st.IsConnected(),
		ProfileName:  st.ProfileName,
		ProxyName:    st.ProxyName,
		ActivityText: st.ActivityText,
	}
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
	p.setBusy(true, "")
	defer p.setBusy(false, "")
	if err := p.App.SimpleConnect(ctx); err != nil {
		p.setBusy(false, err.Error())
		return err
	}
	return p.refresh(ctx)
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

func (p *Presenter) setBusy(busy bool, errText string) {
	p.mu.Lock()
	p.conn.Busy = busy
	if errText != "" {
		p.conn.ErrorText = errText
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
	_ = p.App.Service.Reload(ctx)
	return p.refresh(ctx)
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
