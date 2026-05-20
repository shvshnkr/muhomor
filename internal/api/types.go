package api

import (
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

// ServiceState matches controller.ServiceState.
type ServiceState string

const (
	StateIdle       ServiceState = "Idle"
	StateConnecting ServiceState = "Connecting"
	StateConnected  ServiceState = "Connected"
	StateStopping   ServiceState = "Stopping"
	StateStopped    ServiceState = "Stopped"
)

// ServiceStatus is GET /v1/service/status.
type ServiceStatus struct {
	State              ServiceState `json:"state"`
	Connected          bool         `json:"connected"`
	ProfileID          int64        `json:"profile_id"`
	ProfileName        string       `json:"profile_name"`
	ProxyName          string       `json:"proxy_name"`
	SubscriptionSource string       `json:"subscription_source,omitempty"`
	ActivityText       string       `json:"activity_text,omitempty"`
}

// IsConnected reports active tunnel session.
func (s ServiceStatus) IsConnected() bool {
	return s.State == StateConnected
}

// Settings is GET/PUT /v1/settings.
type Settings struct {
	ServiceMode       string  `json:"service_mode"`
	RouteQuickProfile int     `json:"route_quick_profile"`
	MixedPort         int     `json:"mixed_port"`
	SocksPort         int     `json:"socks_port"`
	HTTPPort          int     `json:"http_port"`
	AllowLAN          bool    `json:"allow_lan"`
	InboundUser       string  `json:"inbound_user"`
	InboundPassword   string  `json:"inbound_password,omitempty"`
	TunEnable         bool    `json:"tun_enable"`
	TunStack          string  `json:"tun_stack"`
	TunDNSHijack      bool    `json:"tun_dns_hijack"`
	TunMTU            int     `json:"tun_mtu"`
	DNSFakeIP         bool    `json:"dns_fake_ip"`
	ChainProfileIDs   []int64 `json:"chain_profile_ids,omitempty"`
}

// Group is GET /v1/groups row.
type Group struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Kind             string `json:"kind"` // subscription | manual
	SubscriptionLink string `json:"subscription_link,omitempty"`
	UserAgent        string `json:"user_agent,omitempty"`      // learned, read-only
	UserAgentMode    string `json:"user_agent_mode,omitempty"` // happ | browser | default | custom
	Builtin          bool   `json:"builtin"`
	ProfileCount     int    `json:"profile_count"`
}

// GroupRequest is POST/PUT /v1/groups.
type GroupRequest struct {
	Name             string `json:"name"`
	Kind             string `json:"kind,omitempty"` // subscription | manual
	SubscriptionLink string `json:"subscription_link,omitempty"`
}

// Profile is one row in GET /v1/profiles.
type Profile struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	URI             string `json:"uri"`
	Enabled         bool   `json:"enabled"`
	LastDelayMs     int    `json:"last_delay_ms"`
	LastError       string `json:"last_error,omitempty"`
	Status          int    `json:"status"`
	Ping            int    `json:"ping"`
	GroupID         int64  `json:"group_id"`
	GroupName       string `json:"group_name,omitempty"`
	WLBuiltinPool   bool   `json:"wl_builtin_pool"`
	WhitelistMarked bool   `json:"whitelist_marked"`
}

// DelayTestResult is POST delay-test response.
type DelayTestResult struct {
	ProfileID int64  `json:"profile_id"`
	DelayMs   int    `json:"delay_ms"`
	Error     string `json:"error,omitempty"`
}

// ImportRequest is POST /v1/profiles/import JSON body.
type ImportRequest struct {
	URI     string   `json:"uri,omitempty"`
	Lines   []string `json:"lines,omitempty"`
	GroupID int64    `json:"group_id,omitempty"`
}

// ImportResult is one imported/skipped line.
type ImportResult struct {
	ID     int64  `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
	Skip   bool   `json:"skip,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// ProfileEnabledRequest is PUT /v1/profiles/{id}/enabled.
type ProfileEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// PingResponse is POST /v1/service/ping.
type PingResponse struct {
	Timestamp   int64  `json:"timestamp"`
	Connected   bool   `json:"connected"`
	ProfileName string `json:"profile_name,omitempty"`
	ProxyName   string `json:"proxy_name,omitempty"`
	DelayMs     int    `json:"delay_ms,omitempty"`
	Error       string `json:"error,omitempty"`
	Path        string `json:"path,omitempty"`
}

// Event is one SSE message (data: json).
type Event struct {
	Type      string         `json:"type"`
	Timestamp int64          `json:"ts"`
	Status    *ServiceStatus `json:"status,omitempty"`
	Reason    string         `json:"reason,omitempty"`
}

func SettingsFromStore(s store.Settings) Settings {
	return Settings{
		ServiceMode:       s.ServiceMode,
		RouteQuickProfile: s.RouteQuickProfile,
		MixedPort:         s.MixedPort,
		SocksPort:         s.SocksPort,
		HTTPPort:          s.HTTPPort,
		AllowLAN:          s.AllowLAN,
		InboundUser:       s.InboundUser,
		InboundPassword:   s.InboundPassword,
		TunEnable:         s.TunEnable,
		TunStack:          s.TunStack,
		TunDNSHijack:      s.TunDNSHijack,
		TunMTU:            s.TunMTU,
		DNSFakeIP:         s.DNSFakeIP,
		ChainProfileIDs:   s.ChainProfileIDs,
	}
}

func (s Settings) ToStore() store.Settings {
	return store.Settings{
		ServiceMode:       s.ServiceMode,
		RouteQuickProfile: s.RouteQuickProfile,
		MixedPort:         s.MixedPort,
		SocksPort:         s.SocksPort,
		HTTPPort:          s.HTTPPort,
		AllowLAN:          s.AllowLAN,
		InboundUser:       s.InboundUser,
		InboundPassword:   s.InboundPassword,
		TunEnable:         s.TunEnable,
		TunStack:          s.TunStack,
		TunDNSHijack:      s.TunDNSHijack,
		TunMTU:            s.TunMTU,
		DNSFakeIP:         s.DNSFakeIP,
		ChainProfileIDs:   s.ChainProfileIDs,
	}
}

func ProfileFromStore(p store.Profile) Profile {
	return Profile{
		ID: p.ID, Name: p.Name, Type: p.Type, URI: p.URI,
		Enabled: p.Enabled, LastDelayMs: p.LastDelayMs, LastError: p.LastError,
		Status: p.Status, Ping: p.Ping, GroupID: p.GroupID,
		WLBuiltinPool: p.WLBuiltinPool, WhitelistMarked: p.WhitelistMarked,
	}
}

func GroupFromStore(g store.Group, count int, builtin bool, ua string) Group {
	return Group{
		ID: g.ID, Name: g.Name, Kind: g.Kind, SubscriptionLink: g.SubscriptionLink,
		UserAgent: ua, UserAgentMode: subscription.UAModeLabel(ua),
		Builtin: builtin, ProfileCount: count,
	}
}
