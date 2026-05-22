package store

import "time"

// bumpProbeOnURLFail updates meta after a failed URL/delay test without resetting cemetery streak to unknown.
func bumpProbeOnURLFail(m ProbeMeta, now time.Time) ProbeMeta {
	m.LastFailAt = now
	m.LastCheckedAt = now
	m.LastErrorClass = "url_test"
	if m.State == ProbeCemetery {
		m.FailStreak++
		m.NextProbeAt = nextProbeAfterFailure(m.State, m.FailStreak, now)
		return m
	}
	m.FailStreak++
	switch {
	case m.FailStreak >= 6:
		m.State = ProbeCemetery
	case m.FailStreak >= 2:
		m.State = ProbeDead
	case m.State == ProbeAlive || m.State == ProbeCandidate:
		m.State = ProbeSuspect
	default:
		if m.State == ProbeUnknown {
			m.State = ProbeSuspect
		}
	}
	m.NextProbeAt = nextProbeAfterFailure(m.State, m.FailStreak, now)
	return m
}

// applyProbeOnURLSuccess marks profile alive after a successful URL test.
func applyProbeOnURLSuccess(m ProbeMeta, delayMs int, now time.Time) ProbeMeta {
	m.State = ProbeAlive
	m.FailStreak = 0
	m.SuccessWindow++
	m.EWMADelayMs = delayMs
	m.LastOKAt = now
	m.LastCheckedAt = now
	m.LastErrorClass = ""
	m.NextProbeAt = now.Add(5 * time.Minute)
	return m
}

// nextProbeAfterFailure mirrors probe.NextProbeAfter (store must not import probe).
func nextProbeAfterFailure(state, failStreak int, now time.Time) time.Time {
	var base time.Duration
	switch state {
	case ProbeSuspect:
		base = 60 * time.Second
	case ProbeDead:
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
	case ProbeCemetery:
		base = 12 * time.Hour
	default:
		return time.Time{}
	}
	jitter := time.Duration(now.UnixNano() % int64(base/7+1))
	return now.Add(base + jitter)
}
