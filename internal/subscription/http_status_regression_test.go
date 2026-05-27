package subscription

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegression_SubscriptionHTTP403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	u := &Updater{Client: srv.Client()}
	_, err := u.fetchSubscription(context.Background(), srv.URL, "muhomor-test")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("err=%q", err.Error())
	}
}

func TestRegression_SubscriptionHTTP402(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Payment Required", http.StatusPaymentRequired)
	}))
	defer srv.Close()

	u := &Updater{Client: srv.Client()}
	_, err := u.fetchSubscription(context.Background(), srv.URL, "muhomor-test")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HTTP 402") {
		t.Fatalf("err=%q", err.Error())
	}
}
