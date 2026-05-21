package store

import "fmt"

// KV keys (DataStore parity subset).
const (
	KeyRouteQuickProfile          = "route_quick_profile"
	KeySelectedProxy              = "selected_proxy"
	KeyAutoSelectFallbackQueue    = "auto_select_fallback_queue"
	KeyAutoSelectFallbackIndex    = "auto_select_fallback_index"
	KeyAutoSelectLastKnownGood    = "auto_select_last_known_good"
	KeySimpleMode                 = "simple_mode"
	KeySimpleModeActivity         = "simple_mode_activity"
	KeyConnectionTestURL          = "connection_test_url"
	KeyConnectionTestTimeoutMs    = "connection_test_timeout_ms"
	KeyActiveWhitelistRestricted  = "active_whitelist_restricted"
	KeySimpleModeUseWLPoolOnly    = "simple_mode_use_wl_pool_only"
	KeyLastBackgroundSubRefreshAt = "simple_mode_last_bg_sub_refresh_at"
	KeyVpnExitIsRussia            = "vpn_exit_is_russia"
	KeyVpnExitProbeProfileID      = "vpn_exit_probe_profile_id"
	KeyProbeSchedulerEnabled    = "probe_scheduler_enabled"
	KeyProbeWarmSelectEnabled   = "probe_warm_select_enabled"
	KeyProbeBuiltinFallbackMaxPct = "probe_builtin_fallback_max_pct"
	KeyProbeSchedulerStats      = "probe_scheduler_stats"
	KeyProbeLastSelectReason    = "probe_last_select_reason"
	KeyProbePreset              = "probe_preset"
)

func KeyGroupUserAgent(groupID int64) string {
	return fmt.Sprintf("group:%d:user_agent", groupID)
}

// Route quick profile values (fr.husi.RouteQuickProfile).
const (
	RouteQuickManual              = 0
	RouteQuickRuDirectOnly        = 1
	RouteQuickRuBlockedAndAIProxy = 2
	RouteQuickWGOverWLTunnel      = 3
)

// Profile status (ProxyEntity subset).
const (
	StatusAvailable   = 0
	StatusInitial     = 1
	StatusUnreachable = 2
)
