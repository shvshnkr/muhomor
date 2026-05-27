package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestListProfilesDueProbeWeighted_afterSeed(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	if err := seedBenchProfiles(ctx, st, 32); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	top, err := st.listProfilesDueProbeByEWMA(ctx, 8, now, 20*time.Minute)
	if err != nil {
		t.Fatalf("ewma: %v", err)
	}
	if len(top) == 0 {
		t.Fatal("expected ewma due profiles")
	}
	fair, err := st.listProfilesDueProbeFair(ctx, 8, now)
	if err != nil {
		t.Fatalf("fair: %v", err)
	}
	out, err := st.ListProfilesDueProbeWeighted(ctx, 8, now, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("expected weighted due profiles")
	}
	_ = fair
}
