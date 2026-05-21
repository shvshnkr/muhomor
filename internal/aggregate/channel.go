package aggregate

import (
	"time"

	"github.com/muhomor/muhomor/internal/store"
)

// ChannelInput is scoring input for one profile/channel.
type ChannelInput struct {
	ProfileID      int64
	WLBuiltin      bool
	RTTMs          int
	URLDelayMs     int
	GoodputKbps    int
	LossPermille   int
	JitterMs       int
	QueueDelayMs   int
	State          int
	Priority       bool
	OnCooldown     bool
}

// ChannelScore lower is better.
type ChannelScore struct {
	ProfileID int64
	Score     int
	Reason    string
}

// ComputeScore ranks a channel using latency + goodput + stability (not RTT alone).
func ComputeScore(in ChannelInput) ChannelScore {
	if in.OnCooldown {
		return ChannelScore{ProfileID: in.ProfileID, Score: 1 << 29, Reason: "cooldown"}
	}
	rtt := in.RTTMs
	if in.URLDelayMs > 0 {
		rtt = in.URLDelayMs
	}
	if rtt <= 0 {
		rtt = 900
	}
	goodput := in.GoodputKbps
	if goodput <= 0 && rtt > 0 {
		goodput = clamp(8000/rtt, 50, 50000)
	}
	if goodput <= 0 {
		goodput = 100
	}
	loss := in.LossPermille
	jitter := in.JitterMs
	queue := in.QueueDelayMs
	// Penalize low goodput with normal RTT (congested narrow pipe).
	score := rtt*2 + (50000/goodput)*3 + loss*4 + jitter*2 + queue*3
	switch in.State {
	case store.ChannelCongested:
		score += 400
	case store.ChannelDegraded:
		score += 1200
	case store.ChannelEmergency:
		score += 5000
	}
	if in.Priority {
		score -= 80
	}
	if in.WLBuiltin {
		score += 200
	}
	reason := "ok"
	if in.State == store.ChannelCongested {
		reason = "congested"
	} else if goodput < 5000 && rtt < 200 {
		reason = "narrow_pipe"
	}
	return ChannelScore{ProfileID: in.ProfileID, Score: score, Reason: reason}
}

// UpdateStateFromSample applies hysteresis to channel tier.
func UpdateStateFromSample(prev int, rttMs, lossPermille, jitterMs int, fail bool) int {
	if fail {
		switch prev {
		case store.ChannelHealthy:
			return store.ChannelCongested
		case store.ChannelCongested:
			return store.ChannelDegraded
		default:
			return store.ChannelEmergency
		}
	}
	if lossPermille > 150 || jitterMs > 120 {
		if prev < store.ChannelCongested {
			return store.ChannelCongested
		}
		return prev
	}
	if rttMs > 0 && rttMs < 250 && lossPermille < 30 {
		return store.ChannelHealthy
	}
	if prev == store.ChannelEmergency {
		return store.ChannelDegraded
	}
	return prev
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// EWMAInt blends sample into previous EWMA.
func EWMAInt(prev, sample, num, den int) int {
	if prev <= 0 {
		return sample
	}
	return (prev*num + sample*den) / (num + den)
}

// GoodputFromDelayKbps estimates kbps from URL/TCP delay ms.
func GoodputFromDelayKbps(delayMs int) int {
	if delayMs <= 0 {
		return 0
	}
	return clamp(8000/delayMs, 50, 50000)
}

// JitterFromSamples returns |a-b| capped.
func JitterFromSamples(prevDelay, newDelay int) int {
	if prevDelay <= 0 || newDelay <= 0 {
		return 0
	}
	d := prevDelay - newDelay
	if d < 0 {
		d = -d
	}
	if d > 500 {
		return 500
	}
	return d
}

// SampleTime returns now for persistence.
func SampleTime() time.Time {
	return time.Now().UTC()
}
