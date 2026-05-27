package wailsapp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/platform"
	"github.com/muhomor/muhomor/internal/store"
)

func (a *App) runCtxOrBG() context.Context {
	if a.runCtx != nil {
		return a.runCtx
	}
	return context.Background()
}

// ListGroups returns subscription groups for the config tab.
func (a *App) ListGroups() ([]GroupDTO, error) {
	if a.core == nil || a.core.Groups == nil {
		return nil, fmt.Errorf("groups API недоступен")
	}
	groups, err := a.core.Groups.ListGroups(a.runCtxOrBG())
	if err != nil {
		return nil, err
	}
	out := make([]GroupDTO, len(groups))
	for i, g := range groups {
		out[i] = GroupDTO{
			ID:               g.ID,
			Name:             g.Name,
			Kind:             g.Kind,
			SubscriptionLink: g.SubscriptionLink,
			Label:            groupLabel(g),
		}
	}
	return out, nil
}

func groupLabel(g apiclient.Group) string {
	name := g.Name
	if name == "" {
		name = fmt.Sprintf("group #%d", g.ID)
	}
	if g.Kind != "" {
		return name + " · " + g.Kind
	}
	return name
}

// ListProfiles returns all profiles.
func (a *App) ListProfiles() ([]ProfileDTO, error) {
	if a.core == nil {
		return nil, fmt.Errorf("app не инициализирован")
	}
	profiles, err := a.core.Config.ListProfiles(a.runCtxOrBG())
	if err != nil {
		return nil, err
	}
	out := make([]ProfileDTO, len(profiles))
	for i, p := range profiles {
		out[i] = ProfileDTO{
			ID:          p.ID,
			Name:        p.Name,
			Type:        p.Type,
			GroupID:     p.GroupID,
			Enabled:     p.Enabled,
			LastDelayMs: p.LastDelayMs,
		}
	}
	return out, nil
}

// GetRouteQuick returns route quick profile index 0–3.
func (a *App) GetRouteQuick() (int, error) {
	if a.core == nil {
		return 0, fmt.Errorf("app не инициализирован")
	}
	return a.core.Config.RouteQuickProfile(a.runCtxOrBG())
}

// SetRouteQuick saves route quick profile 0–3.
func (a *App) SetRouteQuick(v int) error {
	if a.core == nil {
		return fmt.Errorf("app не инициализирован")
	}
	return a.core.Config.SetRouteQuickProfile(a.runCtxOrBG(), v)
}

// LoadSettingsJSON returns daemon settings as JSON-friendly map for the settings tab.
func (a *App) LoadSettingsJSON() (map[string]interface{}, error) {
	if a.core == nil {
		return nil, fmt.Errorf("app не инициализирован")
	}
	set, err := a.core.Config.LoadSettings(a.runCtxOrBG())
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"serviceMode":              set.ServiceMode,
		"mixedPort":                set.MixedPort,
		"routeQuick":               set.RouteQuickProfile,
		"multipathEnabled":         set.MultipathEnabled,
		"multipathPreset":          set.MultipathPreset,
		"multipathWlEmergencyOnly": set.MultipathWLEmergencyOnly,
		"wlBuiltinConnectEnabled":  set.WLBuiltinConnectEnabled,
		"stopDaemonOnExit":         set.StopDaemonOnExit,
		"uiKeepErrorsOnScreen":     set.UIKeepErrorsOnScreen,
		"bulkEnabled":         set.BulkEnabled,
		"bulkLbStrategy":      set.BulkLBStrategy,
		"bulkMinHealthyLegs":  set.BulkMinHealthyLegs,
		"bulkMaxLegs":         set.BulkMaxLegs,
		"bulkRecoverySeconds": set.BulkRecoverySeconds,
		"aggregationMode":     set.AggregationMode,
	}, nil
}

// SaveSettingsJSON persists settings from the settings tab (partial update).
func (a *App) SaveSettingsJSON(patch map[string]interface{}) error {
	if a.core == nil {
		return fmt.Errorf("app не инициализирован")
	}
	set, err := a.core.Config.LoadSettings(a.runCtxOrBG())
	if err != nil {
		return err
	}
	if v, ok := patch["serviceMode"].(string); ok {
		set.ServiceMode = v
		set.TunEnable = v == appcore.ServiceModeVPN
	}
	if v, ok := patch["mixedPort"].(float64); ok {
		set.MixedPort = int(v)
	}
	if v, ok := patch["multipathEnabled"].(bool); ok {
		set.MultipathEnabled = v
	}
	if v, ok := patch["multipathPreset"].(string); ok {
		set.MultipathPreset = v
	}
	if v, ok := patch["multipathWlEmergencyOnly"].(bool); ok {
		set.MultipathWLEmergencyOnly = v
	}
	if v, ok := patch["wlBuiltinConnectEnabled"].(bool); ok {
		set.WLBuiltinConnectEnabled = v
	}
	if v, ok := patch["stopDaemonOnExit"].(bool); ok {
		set.StopDaemonOnExit = v
	}
	if v, ok := patch["uiKeepErrorsOnScreen"].(bool); ok {
		set.UIKeepErrorsOnScreen = v
		if !v && a.pres != nil {
			a.pres.ClearPinnedError()
		}
	}
	if v, ok := patch["bulkEnabled"].(bool); ok {
		set.BulkEnabled = v
	}
	if v, ok := patch["bulkLbStrategy"].(string); ok {
		set.BulkLBStrategy = v
	}
	if v, ok := patch["bulkMinHealthyLegs"].(float64); ok {
		set.BulkMinHealthyLegs = int(v)
	}
	if v, ok := patch["bulkMaxLegs"].(float64); ok {
		set.BulkMaxLegs = int(v)
	}
	if v, ok := patch["bulkRecoverySeconds"].(float64); ok {
		set.BulkRecoverySeconds = int(v)
	}
	if v, ok := patch["aggregationMode"].(string); ok {
		set.AggregationMode = v
	}
	if _, err := a.core.Config.SaveSettings(a.runCtxOrBG(), set); err != nil {
		return err
	}
	if a.pres != nil {
		_ = a.pres.Refresh(a.runCtxOrBG())
	}
	return nil
}

// ReloadMihomo reloads mihomo config when connected.
func (a *App) ReloadMihomo() error {
	if a.core == nil {
		return fmt.Errorf("app не инициализирован")
	}
	return a.core.Service.Reload(a.runCtxOrBG())
}

// BulkPingAll probes all PROXY_BULK legs.
func (a *App) BulkPingAll() (string, error) {
	if a.pres == nil {
		return "", fmt.Errorf("presenter недоступен")
	}
	resp, err := a.pres.BulkPingAll(a.runCtxOrBG())
	if err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", fmt.Errorf("%s", resp.Error)
	}
	return fmt.Sprintf("%d/%d живых", resp.OK, resp.Total), nil
}

// RefreshGroup subscription for selected group id.
func (a *App) RefreshGroup(groupID int64) (string, error) {
	if a.core == nil || a.core.Groups == nil {
		return "", fmt.Errorf("groups API недоступен")
	}
	out, err := a.core.Groups.RefreshGroup(a.runCtxOrBG(), groupID)
	if err != nil {
		return "", err
	}
	if msg, ok := out["message"].(string); ok && msg != "" {
		return msg, nil
	}
	return "Подписка обновлена", nil
}

// UpdateGroupSubscription saves subscription URL for a group.
func (a *App) UpdateGroupSubscription(groupID int64, link string) error {
	if a.core == nil || a.core.Groups == nil {
		return fmt.Errorf("groups API недоступен")
	}
	return a.core.Groups.UpdateGroup(a.runCtxOrBG(), groupID, apiclient.GroupRequest{
		SubscriptionLink: strings.TrimSpace(link),
	})
}

// AddGroupServer adds a server URI to a group.
func (a *App) AddGroupServer(groupID int64, uri string) error {
	if a.core == nil || a.core.Groups == nil {
		return fmt.Errorf("groups API недоступен")
	}
	_, err := a.core.Groups.AddGroupServer(a.runCtxOrBG(), groupID, strings.TrimSpace(uri))
	return err
}

// StopDaemon stops background daemon.
func (a *App) StopDaemon() error {
	return platform.StopDaemon(a.runCtxOrBG(), a.opt.Layout)
}

// StartDaemon ensures daemon is running.
func (a *App) StartDaemon() error {
	args := a.opt.DaemonArgs
	if len(args) == 0 {
		args = defaultDaemonArgs(a.opt)
	}
	return platform.EnsureDaemon(a.runCtxOrBG(), a.opt.Layout, args)
}

// SocketPath returns UDS/socket path for display.
func (a *App) SocketPath() string {
	return a.opt.Layout.SocketPath()
}

// ParseInt helper for frontend numeric fields.
func ParseInt(s string, def int) int {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return v
}

// GroupKinds returns subscription/manual kind strings.
func (a *App) GroupKinds() []string {
	return []string{store.GroupKindSubscription, store.GroupKindManual}
}
