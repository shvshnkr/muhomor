package routing

import (
	"strings"
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestQuickProfile_blockedBeforeRuDirect(t *testing.T) {
	rules := QuickProfileRuleLines(store.RouteQuickRuBlockedAndAIProxy)
	joined := strings.Join(rules, "\n")
	ru := strings.Index(joined, "GEOIP,ru,DIRECT")
	blocked := strings.Index(joined, "geosite-ru-blocked")
	if blocked < 0 || ru < 0 || blocked > ru {
		t.Fatalf("blocked/AI must be before RU direct:\n%s", joined)
	}
}

func TestQuickProfile_wgOverWL_usesProxyForPrivate(t *testing.T) {
	rules := QuickProfileRuleLines(store.RouteQuickWGOverWLTunnel)
	joined := strings.Join(rules, "\n")
	if !strings.Contains(joined, "NETWORK,udp,PROXY") {
		t.Fatalf("expected UDP via proxy in WG-over-WL preset:\n%s", joined)
	}
	if !strings.Contains(joined, "GEOIP,private,PROXY") {
		t.Fatalf("expected private via proxy in WG-over-WL preset:\n%s", joined)
	}
	if strings.Contains(joined, ",DIRECT") {
		t.Fatalf("WG-over-WL preset must not include DIRECT:\n%s", joined)
	}
}
