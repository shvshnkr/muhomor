package store

// KV keys (DataStore parity subset).
const (
	KeyRouteQuickProfile          = "route_quick_profile"
	KeySelectedProxy              = "selected_proxy"
	KeyAutoSelectFallbackQueue    = "auto_select_fallback_queue"
	KeyAutoSelectFallbackIndex    = "auto_select_fallback_index"
	KeyAutoSelectLastKnownGood    = "auto_select_last_known_good"
	KeySimpleMode                 = "simple_mode"
	KeyActiveWhitelistRestricted  = "active_whitelist_restricted"
	KeySimpleModeUseWLPoolOnly    = "simple_mode_use_wl_pool_only"
	KeyLastBackgroundSubRefreshAt = "simple_mode_last_bg_sub_refresh_at"
	KeyVpnExitIsRussia            = "vpn_exit_is_russia"
	KeyVpnExitProbeProfileID      = "vpn_exit_probe_profile_id"
)

// Route quick profile values (fr.husi.RouteQuickProfile).
const (
	RouteQuickManual              = 0
	RouteQuickRuDirectOnly        = 1
	RouteQuickRuBlockedAndAIProxy = 2
)

// Profile status (ProxyEntity subset).
const (
	StatusAvailable   = 0
	StatusInitial     = 1
	StatusUnreachable = 2
)
