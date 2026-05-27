package controller

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRegression_SkipTrafficSample_duringConnect(t *testing.T) {
	r := &Runtime{status: Status{State: StateIdle}}
	ctx := r.beginConnect(context.Background())
	defer r.endConnect()
	_ = ctx
	if !r.skipTrafficSample() {
		t.Fatal("traffic sampling must be skipped while connect in flight")
	}
}

func TestRegression_SkipTrafficSample_connectingState(t *testing.T) {
	r := &Runtime{status: Status{State: StateConnecting}}
	if !r.skipTrafficSample() {
		t.Fatal("traffic sampling must be skipped in StateConnecting")
	}
}

func TestRegression_SkipTrafficSample_reloadGrace(t *testing.T) {
	r := &Runtime{status: Status{State: StateConnected}}
	r.mu.Lock()
	r.mihomoReloadAt = time.Now()
	r.mu.Unlock()
	if !r.skipTrafficSample() {
		t.Fatal("traffic sampling must be skipped shortly after mihomo reload")
	}
}

func TestRegression_SkipTrafficSample_afterReloadGrace(t *testing.T) {
	r := &Runtime{status: Status{State: StateConnected}}
	r.mu.Lock()
	r.mihomoReloadAt = time.Now().Add(-trafficReloadGrace - time.Millisecond)
	r.mu.Unlock()
	if r.skipTrafficSample() {
		t.Fatal("traffic sampling should resume after reload grace window")
	}
}

func TestRegression_QueuedConnectAfterStop(t *testing.T) {
	r := &Runtime{}
	r.queueConnectAfterStop()
	if !r.consumeQueuedConnectAfterStop() {
		t.Fatal("expected queued connect after stop")
	}
	if r.consumeQueuedConnectAfterStop() {
		t.Fatal("queue must be single-use")
	}
}

func TestRegression_StartWhileStopping_queuesConnect(t *testing.T) {
	r := &Runtime{status: Status{State: StateStopping}}
	if !r.isStopping() {
		t.Fatal("fixture must be in stopping state")
	}
	r.queueConnectAfterStop()
	r.mu.Lock()
	queued := r.pendingConnectAfterStop
	r.mu.Unlock()
	if !queued {
		t.Fatal("connect must be queued when service is stopping")
	}
}

func TestRegression_IsTrafficReloadNoise(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("read tcp: wsarecv: An existing connection was forcibly closed"), true},
		{errors.New("connection reset by peer"), true},
		{errors.New("context deadline exceeded"), false},
	}
	for _, tc := range cases {
		if got := isTrafficReloadNoise(tc.err); got != tc.want {
			t.Fatalf("isTrafficReloadNoise(%v)=%v want %v", tc.err, got, tc.want)
		}
	}
}
