package simplemode

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/muhomor/muhomor/internal/selector"
)

const (
	healthInterval = 30 * time.Second
	healthFailLimit = 2
	healthWarmup   = 400 * time.Millisecond
)

// SessionHealth periodic URL/delay check while connected.
type SessionHealth struct {
	Selector   *selector.Selector
	Delay      selector.DelayTester
	OnUnhealthy func(ctx context.Context, profileID int64) error
	Log        *slog.Logger

	mu              sync.Mutex
	cancel          context.CancelFunc
	profileID       int64
	proxyName       string
	consecutiveFail int
}

func (h *SessionHealth) Start(ctx context.Context, profileID int64, proxyName string) {
	h.Stop()
	h.profileID = profileID
	h.proxyName = proxyName
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
	delay, err := h.Delay.TestProxyDelay(ctx, h.proxyName)
	if err == nil && delay > 0 {
		h.consecutiveFail = 0
		return true
	}
	h.consecutiveFail++
	h.Log.Info("session health fail", "profile", h.profileID, "streak", h.consecutiveFail, "event", "H34")
	if h.consecutiveFail >= healthFailLimit && h.OnUnhealthy != nil {
		_ = h.OnUnhealthy(ctx, h.profileID)
		return false
	}
	return true
}
