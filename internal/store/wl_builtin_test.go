package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestWLBuiltinConnect_defaultOff_effectiveCapZero(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "wl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	if st.WLBuiltinConnectEnabled(ctx) {
		t.Fatal("expected default off")
	}
	if st.EffectiveBuiltinFallbackMaxPct(ctx) != 0 {
		t.Fatalf("cap %d", st.EffectiveBuiltinFallbackMaxPct(ctx))
	}
	_ = st.SetKV(ctx, KeyWLBuiltinConnectEnabled, "true")
	_ = st.SetKV(ctx, KeyProbeBuiltinFallbackMaxPct, "25")
	if !st.WLBuiltinConnectEnabled(ctx) {
		t.Fatal("expected on")
	}
	if st.EffectiveBuiltinFallbackMaxPct(ctx) != 25 {
		t.Fatalf("cap %d", st.EffectiveBuiltinFallbackMaxPct(ctx))
	}
}
