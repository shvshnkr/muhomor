package probe

import (
	"math/rand"
	"time"

	"github.com/muhomor/muhomor/internal/store"
)

// NextProbeAfter failure schedules next probe with exp backoff + jitter.
func NextProbeAfter(state int, failStreak int, now time.Time) time.Time {
	var base time.Duration
	switch state {
	case store.ProbeSuspect:
		base = 60 * time.Second
	case store.ProbeDead:
		switch {
		case failStreak <= 1:
			base = 5 * time.Minute
		case failStreak <= 3:
			base = 20 * time.Minute
		case failStreak <= 6:
			base = 2 * time.Hour
		default:
			base = 12 * time.Hour
		}
	case store.ProbeCemetery:
		base = 12 * time.Hour
	default:
		return time.Time{}
	}
	jitter := time.Duration(rand.Int63n(int64(base / 7)))
	return now.Add(base + jitter)
}
