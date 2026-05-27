package simplemode

import (
	"testing"
	"time"
)

func TestRegression_ConnectRefreshBudget_wlVsOpen(t *testing.T) {
	if connectRefreshBudget(true) != 8*time.Second {
		t.Fatal("wl budget")
	}
	if connectRefreshBudget(false) != 2800*time.Millisecond {
		t.Fatal("open network budget")
	}
}
