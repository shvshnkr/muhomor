package simplemode

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestSessionHealth_WindowFailsTriggerUnhealthy(t *testing.T) {
	var calls atomic.Int32
	h := &SessionHealth{
		proxyName: "PROXY",
		Delay:     &stubDelay{ok: false},
		OnUnhealthy: func(ctx context.Context, id int64) error {
			calls.Add(1)
			return nil
		},
	}
	h.profileID = 42
	h.consecutiveFail = 0
	h.recentFails = []time.Time{
		time.Now().Add(-8 * time.Minute),
		time.Now().Add(-4 * time.Minute),
		time.Now().Add(-1 * time.Minute),
	}
	if !healthWouldTrigger(h) {
		t.Fatal("expected window unhealthy with 3 recent fails")
	}
	h.recentFails = append(h.recentFails, time.Now())
	if !healthWouldTrigger(h) {
		t.Fatal("expected unhealthy after 3 fails in window")
	}
}

func healthWouldTrigger(h *SessionHealth) bool {
	now := time.Now()
	cutoff := now.Add(-healthWindowDuration)
	windowFails := 0
	for _, t := range h.recentFails {
		if t.After(cutoff) {
			windowFails++
		}
	}
	return h.consecutiveFail >= healthFailLimit || windowFails >= healthWindowFails
}

type stubDelay struct {
	ok bool
}

func (s *stubDelay) TestProxyDelay(ctx context.Context, proxyName string) (int, error) {
	if s.ok {
		return 100, nil
	}
	return 0, context.DeadlineExceeded
}
