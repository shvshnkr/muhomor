package store

import "time"

// Probe tier states (2K scheme).
const (
	ProbeUnknown   = 0
	ProbeCandidate = 1
	ProbeAlive     = 2
	ProbeSuspect   = 3
	ProbeDead      = 4
	ProbeCemetery  = 5
)

// Probe source priority for fallback diversification.
const (
	ProbeSourceSubscription = 0
	ProbeSourcePinned       = 1
	ProbeSourceBuiltin      = 2
)

// ProbeMeta persisted probe quality (profiles table, migration v5).
type ProbeMeta struct {
	State              int
	LastCheckedAt      time.Time
	LastOKAt           time.Time
	LastFailAt         time.Time
	FailStreak         int
	SuccessWindow      int
	EWMADelayMs        int
	LastErrorClass     string
	NextProbeAt        time.Time
	SourcePriority     int
}

func (m ProbeMeta) IsWarmAlive(maxAge time.Duration) bool {
	if m.State != ProbeAlive && m.State != ProbeCandidate {
		return false
	}
	if m.EWMADelayMs <= 0 && m.State != ProbeAlive {
		return false
	}
	if maxAge > 0 && !m.LastOKAt.IsZero() && time.Since(m.LastOKAt) > maxAge {
		return false
	}
	if maxAge > 0 && !m.LastCheckedAt.IsZero() && time.Since(m.LastCheckedAt) > maxAge*2 {
		return false
	}
	return true
}

func (m ProbeMeta) DueForProbe(now time.Time) bool {
	if m.NextProbeAt.IsZero() {
		return m.State == ProbeUnknown
	}
	return !now.Before(m.NextProbeAt)
}

func ProbeStateName(s int) string {
	switch s {
	case ProbeCandidate:
		return "candidate"
	case ProbeAlive:
		return "alive"
	case ProbeSuspect:
		return "suspect"
	case ProbeDead:
		return "dead"
	case ProbeCemetery:
		return "cemetery"
	default:
		return "unknown"
	}
}

// LegacyStatusToProbe maps old status/ping into initial probe row.
func LegacyStatusToProbe(status, ping int) ProbeMeta {
	m := ProbeMeta{State: ProbeUnknown, SourcePriority: ProbeSourceSubscription}
	switch status {
	case StatusAvailable:
		m.State = ProbeAlive
		if ping > 0 {
			m.EWMADelayMs = ping
		}
	case StatusUnreachable:
		m.State = ProbeDead
		m.FailStreak = 1
	default:
		m.State = ProbeUnknown
	}
	return m
}
