package store

import (
	"context"
	"testing"
)

func TestTryMoveFallbackSkip_skipsDead(t *testing.T) {
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	_ = st.SetFallbackQueue(ctx, []int64{10, 20, 30})
	skip := func(id int64) bool { return id == 20 }
	next, ok := st.TryMoveFallbackSkip(ctx, 10, skip)
	if !ok || next != 30 {
		t.Fatalf("next=%d ok=%v want 30", next, ok)
	}
}
