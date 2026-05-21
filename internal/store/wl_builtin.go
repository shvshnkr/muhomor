package store

import "context"

const KeyWLBuiltinConnectEnabled = "wl_builtin_connect_enabled"

// WLBuiltinConnectEnabled allows H22-retry and builtin in 2K/warm/fallback queues.
// Default false: subscription-only on open network; no free trojan rescue.
func (s *Store) WLBuiltinConnectEnabled(ctx context.Context) bool {
	return boolKV(ctx, s, KeyWLBuiltinConnectEnabled, false)
}

// BuiltinFallbackMaxPct returns builtin share cap; forced 0 when WL builtin connect is off.
func (s *Store) EffectiveBuiltinFallbackMaxPct(ctx context.Context) int {
	if !s.WLBuiltinConnectEnabled(ctx) {
		return 0
	}
	return s.BuiltinFallbackMaxPct(ctx)
}
