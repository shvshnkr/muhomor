package controller

import (
	"testing"
	"time"
)

func TestRegression_RecordRestartAttempt_incrementsInWindow(t *testing.T) {
	r := &Runtime{}
	r.recordRestartAttempt()
	r.recordRestartAttempt()
	r.restartMu.Lock()
	n := r.restartAttempts
	r.restartMu.Unlock()
	if n != 2 {
		t.Fatalf("attempts=%d", n)
	}
}

func TestRegression_RecordRestartAttempt_resetsWindowAfter5Min(t *testing.T) {
	r := &Runtime{}
	r.recordRestartAttempt()
	r.restartMu.Lock()
	r.restartWindowAt = time.Now().Add(-6 * time.Minute)
	r.restartAttempts = 5
	r.restartMu.Unlock()
	r.recordRestartAttempt()
	r.restartMu.Lock()
	n := r.restartAttempts
	r.restartMu.Unlock()
	if n != 1 {
		t.Fatalf("window reset: attempts=%d want 1", n)
	}
}

func TestRegression_BumpDegradedCycle_increments(t *testing.T) {
	r := &Runtime{}
	r.bumpDegradedCycle()
	r.bumpDegradedCycle()
	r.restartMu.Lock()
	n := r.degradedCycles
	r.restartMu.Unlock()
	if n != 2 {
		t.Fatalf("cycles=%d", n)
	}
}
