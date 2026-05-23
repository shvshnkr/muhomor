package simplemode

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/muhomor/muhomor/internal/aggregate"
	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/selector"
	"github.com/muhomor/muhomor/internal/store"
)

const healthBulkDelayWorkers = 12

const (
	healthInterval = 30 * time.Second
	healthFailLimit = 2
	healthWarmup   = 400 * time.Millisecond
)

// SessionHealth periodic URL/delay check while connected.
type SessionHealth struct {
	Selector   *selector.Selector
	Store      *store.Store
	Delay      selector.DelayTester
	Feedback   *aggregate.Feedback
	OnUnhealthy func(ctx context.Context, profileID int64) error
	Log        *slog.Logger

	mu              sync.Mutex
	cancel          context.CancelFunc
	profileID       int64
	proxyName       string
	bulkMemberTags  []string
	consecutiveFail int
}

func (h *SessionHealth) Start(ctx context.Context, profileID int64, proxyName string, bulkMemberTags []string) {
	h.Stop()
	h.profileID = profileID
	h.proxyName = proxyName
	h.bulkMemberTags = append([]string(nil), bulkMemberTags...)
	h.consecutiveFail = 0
	runCtx, cancel := context.WithCancel(ctx)
	h.cancel = cancel
	go h.loop(runCtx)
}

func (h *SessionHealth) Stop() {
	if h.cancel != nil {
		h.cancel()
		h.cancel = nil
	}
	h.consecutiveFail = 0
}

func (h *SessionHealth) loop(ctx context.Context) {
	time.Sleep(healthInterval)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		ok := h.checkOnce(ctx)
		if !ok {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(healthInterval):
		}
	}
}

func (h *SessionHealth) checkOnce(ctx context.Context) bool {
	if h.Delay == nil || h.proxyName == "" {
		return true
	}
	time.Sleep(healthWarmup)
	delay, err := h.probeDelay(ctx)
	if err == nil && delay > 0 {
		h.consecutiveFail = 0
		if h.Feedback != nil {
			_ = h.Feedback.RecordDelaySample(ctx, h.profileID, delay, true)
		}
		return true
	}
	if h.Feedback != nil {
		_ = h.Feedback.RecordDelaySample(ctx, h.profileID, 0, false)
	}
	h.consecutiveFail++
	h.Log.Info("session health fail", "profile", h.profileID, "streak", h.consecutiveFail, "event", "H34")
	if h.consecutiveFail >= healthFailLimit && h.OnUnhealthy != nil {
		_ = h.OnUnhealthy(ctx, h.profileID)
		return false
	}
	return true
}

func (h *SessionHealth) probeDelay(ctx context.Context) (int, error) {
	if h.Delay == nil || h.proxyName == "" {
		return 0, nil
	}
	delay, err := h.Delay.TestProxyDelay(ctx, h.proxyName)
	if err == nil && delay > 0 {
		return delay, nil
	}
	if h.proxyName != configgen.GroupPROXYBulk || len(h.bulkMemberTags) == 0 {
		return delay, err
	}
	best := 0
	var lastErr error
	var mu sync.Mutex
	sem := make(chan struct{}, healthBulkDelayWorkers)
	var wg sync.WaitGroup
	for _, tag := range h.bulkMemberTags {
		tag := tag
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			d, e := h.Delay.TestProxyDelay(ctx, tag)
			if e == nil && d > 0 {
				mu.Lock()
				if best == 0 || d < best {
					best = d
				}
				mu.Unlock()
				return
			}
			if e != nil {
				mu.Lock()
				lastErr = e
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if best > 0 {
		return best, nil
	}
	if lastErr != nil {
		return 0, lastErr
	}
	return delay, err
}
