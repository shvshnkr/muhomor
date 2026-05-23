package selector

import (
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestCompositeScore_prefersURL(t *testing.T) {
	p := store.Profile{ID: 1}
	tcp := map[int64]int{1: 50}
	url := map[int64]int{1: 120}
	if compositeScore(p, tcp, url) != 120 {
		t.Fatalf("expected url latency to win")
	}
}

func TestURLTestCandidates_tcpLiveFirst(t *testing.T) {
	pool := []store.Profile{
		{ID: 1, Name: "dead"},
		{ID: 2, Name: "fast"},
		{ID: 3, Name: "slow"},
	}
	tcp := map[int64]int{2: 10, 3: 200}
	got := urlTestCandidates(pool, tcp, nil, 2)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("order=%v,%v want 2,3", got[0].ID, got[1].ID)
	}
}

func TestCompositeScore_tcpSynthetic(t *testing.T) {
	p := store.Profile{ID: 2}
	tcp := map[int64]int{2: 100}
	s := compositeScore(p, tcp, nil)
	if s < 1000 || s > 2000 {
		t.Fatalf("unexpected synthetic score %d", s)
	}
}
