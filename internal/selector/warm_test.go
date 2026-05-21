package selector

import (
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestApplyBuiltinFallbackCap_limitsBuiltin(t *testing.T) {
	ranked := []store.Profile{
		{ID: 1, WLBuiltinPool: true},
		{ID: 2, WLBuiltinPool: true},
		{ID: 3, WLBuiltinPool: true},
		{ID: 4},
		{ID: 5},
		{ID: 6},
	}
	out := applyBuiltinFallbackCap(ranked, 25)
	builtin := 0
	for _, p := range out {
		if p.WLBuiltinPool {
			builtin++
		}
	}
	if builtin > 2 {
		t.Fatalf("expected at most 2 builtin in 6 (25%%), got %d", builtin)
	}
	if len(out) != 4 {
		t.Fatalf("expected 4 entries (1 builtin + 3 sub), got %d", len(out))
	}
}
