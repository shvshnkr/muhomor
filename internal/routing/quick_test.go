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
