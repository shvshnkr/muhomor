package store

import (
	"context"
	"testing"
	"time"
)

func TestBumpProbeOnURLFail_preservesCemetery(t *testing.T) {
	now := time.Now()
	m := ProbeMeta{State: ProbeCemetery, FailStreak: 6}
	out := bumpProbeOnURLFail(m, now)
	if out.State != ProbeCemetery {
		t.Fatalf("state=%d want cemetery", out.State)
	}
	if out.FailStreak != 7 {
		t.Fatalf("streak=%d want 7", out.FailStreak)
	}
	if out.NextProbeAt.IsZero() {
		t.Fatal("expected next probe scheduled")
	}
}

func TestUpdateProfileProbe_doesNotResetCemeteryToUnknown(t *testing.T) {
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	id, err := st.UpsertProfile(ctx, "cem", "vless", "vless://x")
	if err != nil {
		t.Fatal(err)
	}
	meta := ProbeMeta{State: ProbeCemetery, FailStreak: 8, SourcePriority: ProbeSourceSubscription}
	if err := st.UpdateProfileProbeMeta(ctx, id, meta); err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateProfileProbe(ctx, id, 0, 0, "url test failed"); err != nil {
		t.Fatal(err)
	}
	got, err := st.ProbeMetaByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != ProbeCemetery {
		t.Fatalf("after url fail state=%d want cemetery", got.State)
	}
}

func TestTryMoveFallback_currentNotInQueueUsesIndex(t *testing.T) {
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	_ = st.SetFallbackQueue(ctx, []int64{10, 20, 30})
	_ = st.SetKV(ctx, KeyAutoSelectFallbackIndex, "1")
	next, ok := st.TryMoveFallback(ctx, 99)
	if !ok || next != 20 {
		t.Fatalf("next=%d ok=%v want 20 (resume at index 1)", next, ok)
	}
}
