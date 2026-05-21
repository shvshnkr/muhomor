package aggregate

import (
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestFilterChannelPool_wlEmergencyOnly(t *testing.T) {
	ranked := []store.Profile{
		{ID: 1, WLBuiltinPool: true},
		{ID: 2},
		{ID: 3},
	}
	out := FilterChannelPool(ranked, WLPolicy{EmergencyOnly: true, MaxPct: 25, SubsDegraded: false})
	for _, p := range out {
		if p.WLBuiltinPool {
			t.Fatal("wl in pool when subs ok")
		}
	}
	out2 := FilterChannelPool(ranked, WLPolicy{EmergencyOnly: true, MaxPct: 25, SubsDegraded: true})
	hasWL := false
	for _, p := range out2 {
		if p.WLBuiltinPool {
			hasWL = true
		}
	}
	if !hasWL {
		t.Fatal("expected wl when subs degraded")
	}
}
