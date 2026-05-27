package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/mihomo"
)

func TestPostConnectDelay_bulkGroupFailsMemberOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/proxies/PROXY_BULK/delay":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"message":"An error occurred in the delay test"}`))
		case r.URL.Path == "/proxies/leg_a/delay":
			_ = json.NewEncoder(w).Encode(map[string]int{"delay": 120})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := mihomo.NewClient(mihomo.ClientOptions{
		Controller: strings.TrimPrefix(srv.URL, "http://"),
		Secret:     "secret",
	})
	plan := configgen.BulkPlan{
		Active:      true,
		MatchTarget: configgen.GroupPROXYBulk,
		BulkTags:    []string{"leg_a", "leg_b"},
	}
	_, delay, err := postConnectDelay(context.Background(), nil, c, plan, "leg_a", "http://www.gstatic.com/generate_204", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if delay != 120 {
		t.Fatalf("delay=%d", delay)
	}
}

func TestRegression_PostConnectDelay_retryOn503ThenSuccess(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxies/leg_a/delay" {
			http.NotFound(w, r)
			return
		}
		if hits.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"message":"An error occurred in the delay test"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]int{"delay": 84})
	}))
	defer srv.Close()

	c := mihomo.NewClient(mihomo.ClientOptions{
		Controller: strings.TrimPrefix(srv.URL, "http://"),
		Secret:     "secret",
	})
	plan := configgen.BulkPlan{
		Active:   false,
		BulkTags: nil,
	}
	_, delay, err := postConnectDelay(context.Background(), nil, c, plan, "leg_a", "http://www.gstatic.com/generate_204", 3000)
	if err != nil {
		t.Fatal(err)
	}
	if delay != 84 {
		t.Fatalf("delay=%d", delay)
	}
	if hits.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", hits.Load())
	}
}
