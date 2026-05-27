package controller

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/muhomor/muhomor/internal/mihomo"
)

func TestNormalizeTrafficRate(t *testing.T) {
	t.Run("kbps", func(t *testing.T) {
		got, mode, err := normalizeTrafficRate(8000)
		if err != nil {
			t.Fatal(err)
		}
		if got != 1_000_000 || mode != "kbps" {
			t.Fatalf("got value=%d mode=%s", got, mode)
		}
	})
	t.Run("bytes per sec fallback", func(t *testing.T) {
		raw := int64(200_000_000)
		got, mode, err := normalizeTrafficRate(raw)
		if err != nil {
			t.Fatal(err)
		}
		if got != raw || mode != "bytes_per_sec" {
			t.Fatalf("got value=%d mode=%s", got, mode)
		}
	})
	t.Run("reject impossible", func(t *testing.T) {
		if _, _, err := normalizeTrafficRate(10_000_000_000); err == nil {
			t.Fatal("expected error for impossible value")
		}
	})
}

func TestNormalizeTrafficRateWithTotals_PrefersBytesPerSec(t *testing.T) {
	v, mode, err := normalizeTrafficRateWithTotals(1_651_477, 1_640_000, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if mode != "bytes_per_sec" {
		t.Fatalf("mode=%s", mode)
	}
	if v != 1_651_477 {
		t.Fatalf("value=%d", v)
	}
}

func TestSanitizeTrafficSample_Spike(t *testing.T) {
	up, down := sanitizeTrafficSample(2_000_000, 1_000_000, 500_000_000, 400_000_000)
	if up != 2_000_000 || down != 1_000_000 {
		t.Fatalf("unexpected sanitized values up=%d down=%d", up, down)
	}
}

func TestShouldLogTrafficUnitFallback_idleZero(t *testing.T) {
	if shouldLogTrafficUnitFallback(0, 0, "zero", "zero") {
		t.Fatal("idle zero traffic should not warn")
	}
	if shouldLogTrafficUnitFallback(100, 50, "kbps", "kbps") {
		t.Fatal("kbps modes should not warn")
	}
	if shouldLogTrafficUnitFallback(100, 50, "bytes_per_sec", "kbps") {
		t.Fatal("raw below 1k should not warn")
	}
	if !shouldLogTrafficUnitFallback(2000, 50, "bytes_per_sec", "kbps") {
		t.Fatal("non-kbps with raw>=1k should warn")
	}
}

func TestRefreshTraffic_StaleAfterConsecutiveErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := mihomo.NewClient(mihomo.ClientOptions{Controller: srv.Listener.Addr().String()})
	r := &Runtime{
		Log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		trafficUp:   123,
		trafficDown: 456,
		trafficAt:   time.Now(),
	}
	ctx := context.Background()
	for i := 0; i < trafficErrorStaleAfter; i++ {
		r.refreshTraffic(ctx, client)
	}
	up, down := r.cachedTraffic()
	if up != 0 || down != 0 {
		t.Fatalf("expected stale cache to be cleared, got up=%d down=%d", up, down)
	}
}
