package probe

import (
	"testing"
	"time"

	"github.com/muhomor/muhomor/internal/store"
)

func TestNextProbeAfter_deadBackoffIncreases(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	t1 := NextProbeAfter(store.ProbeDead, 1, now)
	t4 := NextProbeAfter(store.ProbeDead, 8, now)
	if !t4.After(t1) {
		t.Fatalf("cemetery backoff should be later than first dead: %v vs %v", t1, t4)
	}
}
