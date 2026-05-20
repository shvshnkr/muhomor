package appcore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/store"
)

// App bundles service control and local config (Android-ready: inject implementations).
type App struct {
	Layout   paths.Layout
	Service  ServiceControl
	Config   ConfigRepository
	Out      OutputSink
	StatusFn func() ([]byte, error) // optional: read desktop-control-status.txt
}

// NewApp wires default dependencies.
func NewApp(layout paths.Layout, svc ServiceControl, cfg ConfigRepository, out OutputSink) *App {
	return &App{
		Layout:  layout,
		Service: svc,
		Config:  cfg,
		Out:     out,
		StatusFn: func() ([]byte, error) {
			return os.ReadFile(layout.ControlStatusFile())
		},
	}
}

// RunCtlCommand maps Dahusim pseudo-GUI numeric keys to ctl actions.
func (a *App) RunCtlCommand(ctx context.Context, ctl string) error {
	switch ctl {
	case "status":
		return a.doStatus(ctx)
	case "ping":
		return a.doPing(ctx)
	case "start":
		return a.doStart(ctx)
	case "stop":
		return a.doStop(ctx)
	case "reload":
		return a.doReload(ctx)
	case "export-log":
		return a.doExportLog(ctx)
	case "update-check":
		return a.doUpdateCheck(ctx)
	case "update-install":
		return a.doUpdateInstall(ctx)
	default:
		return fmt.Errorf("unknown ctl: %s", ctl)
	}
}

func (a *App) doStatus(ctx context.Context) error {
	st, err := a.Service.Status(ctx)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(st, "", "  ")
	a.Out.Block("API status:\n" + string(b))
	if a.StatusFn != nil {
		if raw, err := a.StatusFn(); err == nil && len(raw) > 0 {
			a.Out.Block("desktop-control-status.txt:\n" + string(raw))
		}
	}
	return nil
}

func (a *App) doPing(ctx context.Context) error {
	st, err := a.Service.Status(ctx)
	if err != nil {
		return err
	}
	pingPath := a.Layout.CacheDir + string(os.PathSeparator) + "desktop-control-ping.txt"
	_ = os.MkdirAll(a.Layout.CacheDir, 0o700)
	body := fmt.Sprintf("timestamp=%d\nstate=%s\nconnected=%t\nprofile=%s\nproxy=%s\n",
		time.Now().UnixMilli(), st.State, st.IsConnected(), st.ProfileName, st.ProxyName)
	if err := os.WriteFile(pingPath, []byte(body), 0o644); err != nil {
		return err
	}
	a.Out.Line("ping OK → " + pingPath)
	if st.IsConnected() {
		a.Out.Line("connected: profile=" + st.ProfileName + " proxy=" + st.ProxyName)
	} else {
		a.Out.Line("not connected (start service to test proxy delay via mihomo)")
	}
	return nil
}

func (a *App) doStart(ctx context.Context) error {
	if err := a.Service.Start(ctx); err != nil {
		return err
	}
	a.Out.Line("start OK (simple mode connect)")
	return a.doStatus(ctx)
}

func (a *App) doStop(ctx context.Context) error {
	if err := a.Service.Stop(ctx); err != nil {
		return err
	}
	a.Out.Line("stop OK")
	return nil
}

func (a *App) doReload(ctx context.Context) error {
	if err := a.Service.Reload(ctx); err != nil {
		return err
	}
	a.Out.Line("reload OK")
	return a.doStatus(ctx)
}

func (a *App) doExportLog(ctx context.Context) error {
	res, err := a.Service.ExportLog(ctx)
	if err != nil {
		return err
	}
	a.Out.Line(formatJSON(res))
	return nil
}

func (a *App) doUpdateCheck(ctx context.Context) error {
	res, err := a.Service.UpdateCheck(ctx)
	if err != nil {
		return err
	}
	a.Out.Line(formatJSON(res))
	return nil
}

func (a *App) doUpdateInstall(ctx context.Context) error {
	res, err := a.Service.UpdateInstall(ctx)
	if err != nil {
		return err
	}
	a.Out.Line(formatJSON(res))
	return nil
}

func formatJSON(m map[string]any) string {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Sprint(m)
	}
	return string(b)
}

// ListProfiles prints all profiles.
func (a *App) ListProfiles(ctx context.Context) error {
	list, err := a.Config.ListProfiles(ctx)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		a.Out.Line("(нет профилей — импортируйте URI)")
		return nil
	}
	var b string
	for _, p := range list {
		b += fmt.Sprintf("  id=%-4d type=%-10s enabled=%-5t wl=%-5t %q\n",
			p.ID, p.Type, p.Enabled, p.WLBuiltinPool, p.Name)
	}
	a.Out.Block("Профили:\n" + b)
	return nil
}

// ImportURI imports one proxy line.
func (a *App) ImportURI(ctx context.Context, uri string) error {
	id, name, typ, err := a.Config.ImportURI(ctx, uri)
	if err != nil {
		return err
	}
	a.Out.Line(fmt.Sprintf("import OK id=%d name=%q type=%s", id, name, typ))
	return nil
}

// RunChain connects relay chain.
func (a *App) RunChain(ctx context.Context, ids []int64) error {
	res, err := a.Service.Chain(ctx, ids)
	if err != nil {
		return err
	}
	a.Out.Line("chain OK " + formatJSON(res))
	return a.doStatus(ctx)
}

// ToggleServiceMode flips proxy ↔ vpn in store.
func (a *App) ToggleServiceMode(ctx context.Context) error {
	set, err := a.Config.LoadSettings(ctx)
	if err != nil {
		return err
	}
	if set.ServiceMode == store.ServiceModeVPN {
		set.ServiceMode = store.ServiceModeProxy
		set.TunEnable = false
	} else {
		set.ServiceMode = store.ServiceModeVPN
		set.TunEnable = true
	}
	if err := a.Config.SaveSettings(ctx, set); err != nil {
		return err
	}
	a.Out.Line(fmt.Sprintf("service_mode=%s (переподключите: [5] reload или [3] start)", set.ServiceMode))
	return nil
}

// SetRouteQuick saves route quick profile.
func (a *App) SetRouteQuick(ctx context.Context, v int) error {
	if err := a.Config.SetRouteQuickProfile(ctx, v); err != nil {
		return err
	}
	labels := map[int]string{
		0: "manual",
		1: "ru_direct",
		2: "ru_blocked_ai",
	}
	a.Out.Line(fmt.Sprintf("route_quick_profile=%d (%s) — применится при следующем connect/reload", v, labels[v]))
	return nil
}

// ShowSettings prints current store settings.
func (a *App) ShowSettings(ctx context.Context) error {
	set, err := a.Config.LoadSettings(ctx)
	if err != nil {
		return err
	}
	qp, _ := a.Config.RouteQuickProfile(ctx)
	a.Out.Block(fmt.Sprintf(`Настройки:
  service_mode=%s tun=%t mixed_port=%d
  route_quick=%d
  inbound_user=%q (password hidden)
  chain_ids=%v
`, set.ServiceMode, set.TunEnable, set.MixedPort, qp, set.InboundUser, set.ChainProfileIDs))
	return nil
}

// Adapt triggers network handoff.
func (a *App) Adapt(ctx context.Context) error {
	if err := a.Service.Adapt(ctx); err != nil {
		return err
	}
	a.Out.Line("adapt scheduled")
	return nil
}

// DaemonHint returns how to start daemon for this layout.
func (a *App) DaemonHint() string {
	return fmt.Sprintf("  muhomor --daemon -d %q", a.Layout.DataDir)
}
