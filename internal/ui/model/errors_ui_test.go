package model

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFriendlyConnectError_contextCanceled(t *testing.T) {
	msg := FriendlyConnectError(context.Canceled)
	if msg != "Подключение отменено" {
		t.Fatalf("msg=%q", msg)
	}
}

func TestFriendlyConnectError_subscriptionDead(t *testing.T) {
	raw := errors.New(`500 Internal Server Error: {"error":"all subscription servers failed probes (WL builtin rescue is off)"}`)
	msg := FriendlyConnectError(raw)
	if !strings.Contains(msg, "WL builtin") {
		t.Fatalf("msg=%q", msg)
	}
	if strings.Contains(msg, "500") {
		t.Fatalf("should not contain HTTP code: %q", msg)
	}
}
