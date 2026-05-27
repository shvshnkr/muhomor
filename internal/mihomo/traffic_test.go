package mihomo

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTrafficNow_firstStreamFrame(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/traffic" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"up":12,"down":34,"upTotal":1,"downTotal":2}`)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// mihomo keeps the socket open; delay handler return without blocking the first frame.
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()

	host := srv.Listener.Addr().String()
	c := NewClient(ClientOptions{Controller: host})
	snap, err := c.TrafficSnapshotNow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Up != 12 || snap.Down != 34 {
		t.Fatalf("got up=%d down=%d", snap.Up, snap.Down)
	}
	if snap.UpTotal != 1 || snap.DownTotal != 2 {
		t.Fatalf("got totals up=%d down=%d", snap.UpTotal, snap.DownTotal)
	}
}
