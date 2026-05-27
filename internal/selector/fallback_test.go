package selector

import (
	"context"
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestCapRankedForFallback_degradedTCPOnly(t *testing.T) {
	ctx := context.Background()
	ranked := make([]store.Profile, 60)
	tcp := map[int64]int{}
	for i := range ranked {
		ranked[i] = store.Profile{ID: int64(i + 1)}
		if i < 50 {
			tcp[ranked[i].ID] = 10 + i
		}
	}
	got := capRankedForFallback(ctx, nil, ranked, tcp, nil, false)
	if len(got) != degradedFallbackMax {
		t.Fatalf("len=%d want %d", len(got), degradedFallbackMax)
	}
	if got[0].ID != 1 {
		t.Fatalf("first id=%d want 1 (best TCP order preserved)", got[0].ID)
	}
}

func TestCapRankedForFallback_withURLUnchanged(t *testing.T) {
	ctx := context.Background()
	ranked := []store.Profile{{ID: 1}, {ID: 2}}
	url := map[int64]int{1: 100}
	got := capRankedForFallback(ctx, nil, ranked, nil, url, false)
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
}
