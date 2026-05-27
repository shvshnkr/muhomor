package standby

import (
	"context"
	"testing"
	"time"

	"github.com/muhomor/muhomor/internal/store"
)

func TestIsHotFresh(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	now := time.Now().UTC()
	entries := []Entry{
		{ProfileID: 1, DelayMs: 100, VerifiedAt: now},
		{ProfileID: 2, DelayMs: 110, VerifiedAt: now},
		{ProfileID: 3, DelayMs: 120, VerifiedAt: now},
	}
	if err := SaveHot(ctx, st, entries); err != nil {
		t.Fatal(err)
	}
	if !IsHotFresh(ctx, st) {
		t.Fatal("expected hot fresh")
	}
	stale := []Entry{{ProfileID: 9, DelayMs: 50, VerifiedAt: now.Add(-HotMaxAge - time.Minute)}}
	_ = SaveHot(ctx, st, stale)
	if IsHotFresh(ctx, st) {
		t.Fatal("expected hot not fresh")
	}
}

func TestApplyDelayResultsHotWarm(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ranked := make([]Entry, 10)
	for i := range ranked {
		ranked[i] = Entry{ProfileID: int64(i + 1), DelayMs: (i + 1) * 10}
	}
	ApplyDelayResults(ctx, st, ranked)
	hot := LoadHot(ctx, st)
	if len(hot) != HotSize {
		t.Fatalf("hot len=%d want %d", len(hot), HotSize)
	}
	warm := LoadWarm(ctx, st)
	wantWarm := len(ranked) - HotSize
	if len(warm) > WarmSize {
		wantWarm = WarmSize
	}
	if len(warm) != wantWarm {
		t.Fatalf("warm len=%d want %d", len(warm), wantWarm)
	}
}

func TestShortFallbackQueue(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	now := time.Now().UTC()
	_ = SaveHot(ctx, st, []Entry{{ProfileID: 10, DelayMs: 80, VerifiedAt: now}})
	_ = SaveWarm(ctx, st, []Entry{{ProfileID: 20, DelayMs: 90, VerifiedAt: now}})
	q := ShortFallbackQueue(ctx, st, 10)
	if len(q) < 1 {
		t.Fatal("expected fallback ids")
	}
	for _, id := range q {
		if id == 10 {
			t.Fatal("active id should be excluded")
		}
	}
}
