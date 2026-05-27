package apiclient

import (
	"errors"
	"strings"
	"testing"
)

func TestRegression_WrapDaemonErr_refusedOnly(t *testing.T) {
	refused := wrapDaemonErr(errors.New("dial tcp 127.0.0.1:8751: connectex: No connection could be made"))
	if refused == nil || !strings.Contains(refused.Error(), "daemon not running") {
		t.Fatalf("expected daemon not running wrap, got %v", refused)
	}
	timeout := wrapDaemonErr(errors.New("context deadline exceeded"))
	if strings.Contains(timeout.Error(), "daemon not running") {
		t.Fatalf("timeout should not be wrapped: %v", timeout)
	}
}
