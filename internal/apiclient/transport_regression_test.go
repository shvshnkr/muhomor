package apiclient

import (
	"errors"
	"testing"
	"time"
)

func TestRegression_InitHTTP_sharedTransportOnce(t *testing.T) {
	c := &Client{}
	t1 := c.sharedTransport()
	t2 := c.sharedTransport()
	if t1 != t2 {
		t.Fatal("sharedTransport must return same instance")
	}
	api := c.apiHTTP()
	sse := c.sseHTTP()
	if api.Transport != sse.Transport {
		t.Fatal("api and sse clients must share transport")
	}
	if api.Timeout != 120*time.Second {
		t.Fatalf("api timeout=%s", api.Timeout)
	}
	if sse.Timeout != 0 {
		t.Fatalf("sse timeout=%s want 0", sse.Timeout)
	}
}

func TestRegression_DaemonUnreachable_refusedMessage(t *testing.T) {
	err := errors.New("dial tcp 127.0.0.1:8751: connectex: No connection could be made")
	if !daemonUnreachable(err) {
		t.Fatal("expected refused")
	}
}
