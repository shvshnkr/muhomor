package selector

import (
	"context"

	"github.com/muhomor/muhomor/internal/store"
)

const degradedFallbackMax = 48

// capRankedForFallback limits fallback queue when URL tests failed (TCP-only ranking is unreliable).
func capRankedForFallback(ranked []store.Profile, tcp, url map[int64]int) []store.Profile {
	if len(url) > 0 || len(ranked) == 0 {
		return ranked
	}
	var live []store.Profile
	for _, p := range ranked {
		if tcp[p.ID] > 0 {
			live = append(live, p)
		}
	}
	if len(live) == 0 {
		if len(ranked) > degradedFallbackMax {
			return ranked[:degradedFallbackMax]
		}
		return ranked
	}
	if len(live) > degradedFallbackMax {
		live = live[:degradedFallbackMax]
	}
	return live
}

func (s *Selector) shouldSkipFallback(ctx context.Context, id int64) bool {
	if s.isCooldown(id) {
		return true
	}
	if s.Store == nil {
		return false
	}
	meta, err := s.Store.ProbeMetaByID(ctx, id)
	if err != nil {
		return false
	}
	switch meta.State {
	case store.ProbeCemetery, store.ProbeDead:
		return true
	default:
		return false
	}
}

func (s *Selector) markProfileFailed(ctx context.Context, id int64) {
	s.RecordFailure(id)
	if s.Store == nil {
		return
	}
	_ = s.Store.UpdateProfileProbe(ctx, id, 0, store.StatusUnreachable, "session_fail")
}
