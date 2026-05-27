package controller

import "testing"

func TestQueuedConnectAfterStopToggle(t *testing.T) {
	r := &Runtime{}
	if r.consumeQueuedConnectAfterStop() {
		t.Fatal("queue should be empty")
	}
	r.queueConnectAfterStop()
	if !r.consumeQueuedConnectAfterStop() {
		t.Fatal("expected queued connect")
	}
	if r.consumeQueuedConnectAfterStop() {
		t.Fatal("queue should be consumed once")
	}
}

