package simplemode

import (
	"context"
	"log/slog"
	"time"

	"github.com/muhomor/muhomor/internal/reachability"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

const (
	settleDelayMs              = 45_000
	backgroundSubRefreshInterval = 45 * 60 * 1000
	wlPostConnectLatencyMaxMs  = 800
)

// Maintenance runs post-connect subscription refresh with guards (RU-OPTIMIZATION).
type Maintenance struct {
	Store   *store.Store
	Updater *subscription.Updater
	Log     *slog.Logger
}

func (m *Maintenance) ScheduleAfterConnect(ctx context.Context, profileID int64, postDelay int, probe reachability.Result) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(settleDelayMs * time.Millisecond):
		}
		m.runRefresh(ctx, profileID, postDelay, probe)
	}()
}

func (m *Maintenance) runRefresh(ctx context.Context, profileID int64, postDelay int, probe reachability.Result) {
	lastStr, _ := m.Store.GetKV(ctx, store.KeyLastBackgroundSubRefreshAt)
	if lastStr != "" {
		var last int64
		if _, err := parseInt64(lastStr, &last); err == nil {
			if time.Now().UnixMilli()-last < backgroundSubRefreshInterval {
				m.Log.Info("bg sub refresh skipped interval", "profile", profileID, "event", "H29")
				return
			}
		}
	}
	wl, _ := m.Store.GetKV(ctx, store.KeyActiveWhitelistRestricted)
	wlOnly := wl == "true"
	if wlOnly && !whitelistConfident(postDelay, probe) {
		m.Log.Info("bg sub refresh skipped wl not confident", "event", "H29")
		return
	}
	if m.Updater != nil {
		if err := m.Updater.RefreshDue(ctx, probe.AnyReachable()); err != nil {
			m.Log.Warn("bg sub refresh", "err", err)
			return
		}
	}
	_ = m.Store.SetKV(ctx, store.KeyLastBackgroundSubRefreshAt, formatInt64(time.Now().UnixMilli()))
}

func whitelistConfident(postDelay int, probe reachability.Result) bool {
	if postDelay > 0 && postDelay <= wlPostConnectLatencyMaxMs {
		return true
	}
	return probe.WhitelistSourceReachable
}

func parseInt64(s string, out *int64) (int, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int64(c-'0')
	}
	*out = n
	return 1, nil
}

func formatInt64(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
