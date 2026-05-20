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

func TestCompositeScore_tcpSynthetic(t *testing.T) {
	p := store.Profile{ID: 2}
	tcp := map[int64]int{2: 100}
	s := compositeScore(p, tcp, nil)
	if s < 1000 || s > 2000 {
		t.Fatalf("unexpected synthetic score %d", s)
	}
}
