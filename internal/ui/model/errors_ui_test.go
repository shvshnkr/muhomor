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

func TestRegression_FriendlyDelayError_503(t *testing.T) {
	msg := FriendlyDelayError(`503 Service Unavailable: {"message":"An error occurred in the delay test"}`)
	if msg != "Сервер временно недоступен" {
		t.Fatalf("msg=%q", msg)
	}
}

func TestFriendlyConnectError_postConnect(t *testing.T) {
	raw := errors.New("post-connect url test failed: 503 Service Unavailable")
	msg := FriendlyConnectError(raw)
	if msg != "Проверка соединения не прошла" && msg != "Сервер временно недоступен" {
		t.Fatalf("msg=%q", msg)
	}
}

func TestRegression_FriendlyConnectError_subscriptionHTTP403(t *testing.T) {
	raw := errors.New(`subscription HTTP 403 from https://example.com/sub (ua=default; body: "Forbidden")`)
	msg := FriendlyConnectError(raw)
	if strings.Contains(msg, "HTTP 403") {
		t.Fatalf("UI should not show raw HTTP code: %q", msg)
	}
	if !strings.Contains(msg, "провайдер") {
		t.Fatalf("expected provider-facing copy: %q", msg)
	}
}

func TestRegression_FriendlyConnectError_subscriptionHTTP402(t *testing.T) {
	raw := errors.New(`subscription HTTP 402 from https://example.com/sub (ua=default; body: "pay")`)
	msg := FriendlyConnectError(raw)
	if strings.Contains(msg, "HTTP 402") {
		t.Fatalf("UI should not show raw HTTP code: %q", msg)
	}
}

func TestRegression_FriendlyConnectError_subscriptionDead(t *testing.T) {
	raw := errors.New(`500 Internal Server Error: {"error":"all subscription servers failed probes (WL builtin rescue is off)"}`)
	msg := FriendlyConnectError(raw)
	if !strings.Contains(msg, "WL builtin") {
		t.Fatalf("msg=%q", msg)
	}
	if strings.Contains(msg, "500") {
		t.Fatalf("should not contain HTTP code: %q", msg)
	}
}
