package mihomo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestUpdateGeoDatabaseIfStale_cached(t *testing.T) {
	dir := t.TempDir()
	path := GeoDatabasePath(dir)
	if err := os.WriteFile(path, make([]byte, geoDBMinBytes), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := UpdateGeoDatabaseIfStale(ctx, dir, 7*24*time.Hour); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadGeoDatabase_http(t *testing.T) {
	body := make([]byte, geoDBMinBytes+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	old := DefaultGeoMMDBURL
	DefaultGeoMMDBURL = srv.URL
	defer func() { DefaultGeoMMDBURL = old }()

	dir := t.TempDir()
	if err := DownloadGeoDatabase(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	if !GeoDatabaseCached(dir) {
		t.Fatal("expected cached geo db")
	}
}
