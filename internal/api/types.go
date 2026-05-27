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

type VerificationPhase string

const (
	VerificationUnknown        VerificationPhase = "unknown"
	VerificationTransportAlive VerificationPhase = "transport_alive"
	VerificationQualityOK      VerificationPhase = "quality_verified"
	VerificationNoLiveServers  VerificationPhase = "no_live_servers"
)

type LiveServerEvidence struct {
	TCPOK            int    `json:"tcp_ok"`
	URLOK            int    `json:"url_ok"`
	PoolSize         int    `json:"pool_size,omitempty"`
	Degraded         bool   `json:"degraded"`
	LastSuccessAtUTC string `json:"last_success_at_utc,omitempty"`
}

// StandbyEntry is one hot/warm standby proxy in service status.
type StandbyEntry struct {
	ProfileID  int64  `json:"profile_id"`
	ProxyName  string `json:"proxy_name,omitempty"`
	DelayMs    int    `json:"delay_ms"`
	VerifiedAt string `json:"verified_at,omitempty"`
}

// StandbyProgress is hot/warm standby pool snapshot (StandbyPool B+C).
type StandbyProgress struct {
	Hot             []StandbyEntry `json:"hot,omitempty"`
	Warm            []StandbyEntry `json:"warm,omitempty"`
	PickerRunning   bool           `json:"picker_running"`
	LastRefreshAt   string         `json:"last_refresh_at,omitempty"`
}

// ProbeProgress is background 2K probe scheduler snapshot (Phase 5).
type ProbeProgress struct {
	TotalEnabled      int    `json:"total_enabled"`
	Alive             int    `json:"alive"`
	Candidate         int    `json:"candidate"`
	Suspect           int    `json:"suspect"`
	Dead              int    `json:"dead"`
	Cemetery          int    `json:"cemetery"`
	LastTickChecked   int    `json:"last_tick_checked"`
	LastTickOK        int    `json:"last_tick_ok"`
	LastTickFail      int    `json:"last_tick_fail"`
	SchedulerEnabled  bool   `json:"scheduler_enabled"`
	WarmSelectEnabled bool   `json:"warm_select_enabled"`
	Preset            string `json:"preset,omitempty"`
	LastSelectReason  string `json:"last_select_reason,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

// BulkMemberStatus is one leg of PROXY_BULK (tag + optional profile + last ping).
type BulkMemberStatus struct {
	Tag       string `json:"tag"`
	ProfileID int64  `json:"profile_id,omitempty"`
	Name      string `json:"name,omitempty"`
	DelayMs   int    `json:"delay_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

// BulkPingAllResponse is POST /v1/service/bulk-ping-all.
type BulkPingAllResponse struct {
	Timestamp int64              `json:"timestamp"`
	Connected bool               `json:"connected"`
	OK        int                  `json:"ok"`
	Total     int                  `json:"total"`
	Members   []BulkMemberStatus   `json:"members"`
	Error     string               `json:"error,omitempty"`
}

// ServiceStatus is GET /v1/service/status.
type ServiceStatus struct {
	State              ServiceState   `json:"state"`
	Connected          bool           `json:"connected"`
	ConnectedVerified  bool           `json:"connected_verified,omitempty"`
	ConnectedDegraded  bool           `json:"connected_degraded,omitempty"`
	LiveServersConfirmed bool         `json:"live_servers_confirmed,omitempty"`
	VerificationPhase  VerificationPhase `json:"verification_phase,omitempty"`
	VerificationReason string         `json:"verification_reason,omitempty"`
	LiveServerEvidence *LiveServerEvidence `json:"live_server_evidence,omitempty"`
	ProfileID          int64          `json:"profile_id"`
	ProfileName        string         `json:"profile_name"`
	ProxyName          string         `json:"proxy_name"`
	SubscriptionSource string         `json:"subscription_source,omitempty"`
	ActivityText       string         `json:"activity_text,omitempty"`
	LastPingMs         int            `json:"last_ping_ms,omitempty"`
	LastPingError      string         `json:"last_ping_error,omitempty"`
	Probe              *ProbeProgress     `json:"probe,omitempty"`
	Standby            *StandbyProgress   `json:"standby,omitempty"`
	Multipath          *MultipathProgress `json:"multipath,omitempty"`
	BulkMembers        []BulkMemberStatus `json:"bulk_members,omitempty"`
	TrafficUp          int64              `json:"traffic_up,omitempty"`
	TrafficDown        int64              `json:"traffic_down,omitempty"`
}

// MultipathProgress desktop channel aggregation snapshot.
type MultipathProgress struct {
	Enabled         bool   `json:"enabled"`
	Preset          string `json:"preset,omitempty"`
	WLEmergencyOnly bool   `json:"wl_emergency_only"`
	ActiveChannels  int    `json:"active_channels"`
	HealthyChannels int    `json:"healthy_channels"`
	LastReason      string `json:"last_reason,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
	AggregationMode    string `json:"aggregation_mode,omitempty"`
	BulkEnabled        bool   `json:"bulk_enabled"`
	BulkActive         bool   `json:"bulk_active"`
	BulkMemberCount    int    `json:"bulk_member_count"`
	BulkMinLegs        int    `json:"bulk_min_legs,omitempty"`
	BulkMaxLegs        int    `json:"bulk_max_legs,omitempty"`
	BulkFallbackReason string `json:"bulk_fallback_reason,omitempty"`
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
	DNSFakeIP                bool   `json:"dns_fake_ip"`
	ChainProfileIDs          []int64 `json:"chain_profile_ids,omitempty"`
	MultipathEnabled         bool   `json:"multipath_enabled"`
	MultipathPreset          string `json:"multipath_preset,omitempty"`
	MultipathWLEmergencyOnly bool   `json:"multipath_wl_emergency_only"`
	WLBuiltinConnectEnabled  bool   `json:"wl_builtin_connect_enabled"`
	StopDaemonOnExit         bool   `json:"stop_daemon_on_exit"`
	UIKeepErrorsOnScreen     bool   `json:"ui_keep_errors_on_screen"`
	AggregationMode          string `json:"aggregation_mode,omitempty"`
	BulkEnabled              bool   `json:"bulk_enabled"`
	BulkMinHealthyLegs       int    `json:"bulk_min_healthy_legs,omitempty"`
	BulkMaxLegs              int    `json:"bulk_max_legs,omitempty"`
	BulkFallbackMode         string `json:"bulk_fallback_mode,omitempty"`
	BulkRecoverySeconds      int    `json:"bulk_recovery_seconds,omitempty"`
	BulkLBStrategy           string `json:"bulk_lb_strategy,omitempty"`
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
		ServiceMode:              s.ServiceMode,
		RouteQuickProfile:        s.RouteQuickProfile,
		MixedPort:                s.MixedPort,
		SocksPort:                s.SocksPort,
		HTTPPort:                 s.HTTPPort,
		AllowLAN:                 s.AllowLAN,
		InboundUser:              s.InboundUser,
		InboundPassword:          s.InboundPassword,
		TunEnable:                s.TunEnable,
		TunStack:                 s.TunStack,
		TunDNSHijack:             s.TunDNSHijack,
		TunMTU:                   s.TunMTU,
		DNSFakeIP:                s.DNSFakeIP,
		ChainProfileIDs:          s.ChainProfileIDs,
		MultipathEnabled:         s.MultipathEnabled,
		MultipathPreset:          s.MultipathPreset,
		MultipathWLEmergencyOnly: s.MultipathWLEmergencyOnly,
		WLBuiltinConnectEnabled:  s.WLBuiltinConnectEnabled,
		StopDaemonOnExit:         s.StopDaemonOnExit,
		UIKeepErrorsOnScreen:     s.UIKeepErrorsOnScreen,
		AggregationMode:          s.AggregationMode,
		BulkEnabled:              s.BulkEnabled,
		BulkMinHealthyLegs:       s.BulkMinHealthyLegs,
		BulkMaxLegs:              s.BulkMaxLegs,
		BulkFallbackMode:         s.BulkFallbackMode,
		BulkRecoverySeconds:      s.BulkRecoverySeconds,
		BulkLBStrategy:           s.BulkLBStrategy,
	}
}

func (s Settings) ToStore() store.Settings {
	return store.Settings{
		ServiceMode:              s.ServiceMode,
		RouteQuickProfile:        s.RouteQuickProfile,
		MixedPort:                s.MixedPort,
		SocksPort:                s.SocksPort,
		HTTPPort:                 s.HTTPPort,
		AllowLAN:                 s.AllowLAN,
		InboundUser:              s.InboundUser,
		InboundPassword:          s.InboundPassword,
		TunEnable:                s.TunEnable,
		TunStack:                 s.TunStack,
		TunDNSHijack:             s.TunDNSHijack,
		TunMTU:                   s.TunMTU,
		DNSFakeIP:                s.DNSFakeIP,
		ChainProfileIDs:          s.ChainProfileIDs,
		MultipathEnabled:         s.MultipathEnabled,
		MultipathPreset:          s.MultipathPreset,
		MultipathWLEmergencyOnly: s.MultipathWLEmergencyOnly,
		WLBuiltinConnectEnabled:  s.WLBuiltinConnectEnabled,
		StopDaemonOnExit:         s.StopDaemonOnExit,
		UIKeepErrorsOnScreen:     s.UIKeepErrorsOnScreen,
		AggregationMode:          s.AggregationMode,
		BulkEnabled:              s.BulkEnabled,
		BulkMinHealthyLegs:       s.BulkMinHealthyLegs,
		BulkMaxLegs:              s.BulkMaxLegs,
		BulkFallbackMode:         s.BulkFallbackMode,
		BulkRecoverySeconds:      s.BulkRecoverySeconds,
		BulkLBStrategy:           s.BulkLBStrategy,
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
