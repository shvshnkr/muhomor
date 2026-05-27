package model

import "testing"

func TestProxyDisplayLabel(t *testing.T) {
	got := ProxyDisplayLabel("🇷🇺 Russia | MIFA", "🇷🇺__Russia__[19]__MIFA_p56")
	if got != "🇷🇺 Russia | MIFA" {
		t.Fatalf("got %q", got)
	}
	if ProxyDisplayLabel("", "tag_p1") != "tag_p1" {
		t.Fatal("fallback to proxy tag")
	}
}
