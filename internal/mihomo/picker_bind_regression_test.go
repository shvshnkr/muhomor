package mihomo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/muhomor/muhomor/internal/paths"
)

func TestRegression_IsPickerBindError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("listen tcp 127.0.0.1:8760: bind: Only one usage of each socket address"), true},
		{errors.New("address already in use"), true},
		{errors.New("context deadline exceeded"), false},
	}
	for _, tc := range cases {
		if got := isPickerBindError(tc.err); got != tc.want {
			t.Fatalf("isPickerBindError(%v)=%v want %v", tc.err, got, tc.want)
		}
	}
}

func TestRegression_ProbePickerAPI_nilWhenVersionFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")
	if c := probePickerAPI(context.Background(), addr, "picker"); c != nil {
		t.Fatal("expected nil client when version fails")
	}
}

func TestRegression_PickerAttachExisting_reusesLiveAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/version" {
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "test"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	ctl := strings.TrimPrefix(srv.URL, "http://")

	dir := t.TempDir()
	p := NewPicker(paths.Layout{RuntimeDir: dir}, "")
	if !p.attachExisting(context.Background(), ctl, "picker") {
		t.Fatal("attachExisting should succeed")
	}
	if !p.Available() {
		t.Fatal("picker should be running after attach")
	}
}
