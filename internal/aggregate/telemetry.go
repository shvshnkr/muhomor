package aggregate

import (
	"context"

	"github.com/muhomor/muhomor/internal/store"
)

// Feedback records a health/probe sample into channel metrics.
type Feedback struct {
	Store *store.Store
}

// RecordDelaySample updates EWMA from successful delay test (ms).
func (f *Feedback) RecordDelaySample(ctx context.Context, profileID int64, delayMs int, ok bool) error {
	if f.Store == nil {
		return nil
	}
	m, err := f.Store.ChannelMetricsByID(ctx, profileID)
	if err != nil {
		return err
	}
	now := SampleTime()
	goodput := GoodputFromDelayKbps(delayMs)
	jitter := JitterFromSamples(m.EWMAQueueDelayMs, delayMs)
	if ok && delayMs > 0 {
		m.EWMAGoodputKbps = EWMAInt(m.EWMAGoodputKbps, goodput, 7, 3)
		m.EWMAJitterMs = EWMAInt(m.EWMAJitterMs, jitter, 7, 3)
		m.EWMAQueueDelayMs = EWMAInt(m.EWMAQueueDelayMs, delayMs, 7, 3)
		m.EWMALossPermille = EWMAInt(m.EWMALossPermille, 0, 9, 1)
		m.State = UpdateStateFromSample(m.State, delayMs, m.EWMALossPermille, m.EWMAJitterMs, false)
	} else {
		m.EWMALossPermille = EWMAInt(m.EWMALossPermille, 200, 7, 3)
		if m.EWMALossPermille > 1000 {
			m.EWMALossPermille = 1000
		}
		m.State = UpdateStateFromSample(m.State, delayMs, m.EWMALossPermille, m.EWMAJitterMs, true)
	}
	m.LastSampleAt = now
	return f.Store.UpdateChannelMetrics(ctx, profileID, m)
}

// BuildInputs builds ChannelInput slice from profiles and maps.
func BuildInputs(profiles []store.Profile, tcp, url map[int64]int, priority map[int64]struct{}, cooldown func(int64) bool, metrics func(int64) store.ChannelMetrics) []ChannelInput {
	out := make([]ChannelInput, 0, len(profiles))
	for _, p := range profiles {
		ch := store.ChannelMetrics{}
		if metrics != nil {
			ch = metrics(p.ID)
		}
		_, pri := priority[p.ID]
		out = append(out, ChannelInput{
			ProfileID:    p.ID,
			WLBuiltin:    p.WLBuiltinPool,
			RTTMs:        tcp[p.ID],
			URLDelayMs:   url[p.ID],
			GoodputKbps:  ch.EWMAGoodputKbps,
			LossPermille: ch.EWMALossPermille,
			JitterMs:     ch.EWMAJitterMs,
			QueueDelayMs: ch.EWMAQueueDelayMs,
			State:        ch.State,
			Priority:     pri,
			OnCooldown:   cooldown != nil && cooldown(p.ID),
		})
	}
	return out
}

// CountHealthy channels in scores below congested threshold.
func CountHealthy(scores []ChannelScore) int {
	n := 0
	for _, sc := range scores {
		if sc.Score < 2000 && sc.Reason != "cooldown" {
			n++
		}
	}
	return n
}
