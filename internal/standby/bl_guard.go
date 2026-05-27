package standby

import (
	"context"

	"github.com/muhomor/muhomor/internal/profileclass"
	"github.com/muhomor/muhomor/internal/store"
)

// blEligible reports whether a standby entry may be used on open network.
func blEligible(ctx context.Context, st *store.Store, p store.Profile, e Entry, wlOnly bool) bool {
	if wlOnly || st == nil || !st.BLExitFilterEnabled(ctx) {
		return true
	}
	if st.BLExitFilterScope(ctx) != store.BLExitScopeAll && !profileclass.IsBLModeUplink(p) {
		return true
	}
	if e.BLDelayMs > 0 {
		return true
	}
	if _, ok := st.BLExitOK(ctx, p.ID); ok {
		return true
	}
	return false
}
