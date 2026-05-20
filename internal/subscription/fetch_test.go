package subscription

import (
	"strings"
	"testing"
)

func TestNormalizeSubscriptionBody_base64(t *testing.T) {
	raw := "dmxlc3M6Ly90ZXN0QDEyNy4wLjAuMTo0NDM="
	out := NormalizeSubscriptionBody([]byte(raw))
	if !strings.Contains(string(out), "vless://") {
		t.Fatalf("expected decoded vless, got %q", out)
	}
}
