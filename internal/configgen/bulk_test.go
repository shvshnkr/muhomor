package configgen

import (
	"strings"
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestResolveBulkPlan_legacy(t *testing.T) {
	set := store.Settings{AggregationMode: store.AggregationModeLegacy}
	plan := ResolveBulkPlan(set, []string{"a", "b"}, "a")
	if plan.Active {
		t.Fatal("expected inactive in legacy")
	}
	if plan.MatchTarget != GroupPROXY {
		t.Fatalf("match=%s", plan.MatchTarget)
	}
}

func TestResolveBulkPlan_active(t *testing.T) {
	set := store.Settings{
		AggregationMode:    store.AggregationModeFlowAggregate,
		BulkEnabled:        true,
		BulkMinHealthyLegs: 2,
		BulkLBStrategy:     store.BulkLBStickySessions,
	}
	plan := ResolveBulkPlan(set, []string{"a_p1", "b_p2"}, "a_p1")
	if !plan.Active {
		t.Fatal("expected active")
	}
	if plan.MatchTarget != GroupPROXYBulk {
		t.Fatalf("match=%s", plan.MatchTarget)
	}
}

func TestProfileIDFromBulkTag(t *testing.T) {
	id, ok := ProfileIDFromBulkTag("trojan_p42")
	if !ok || id != 42 {
		t.Fatalf("got id=%d ok=%v", id, ok)
	}
	if _, ok := ProfileIDFromBulkTag("noid"); ok {
		t.Fatal("expected false")
	}
}

func TestResolveBulkPlan_notEnoughLegs(t *testing.T) {
	set := store.Settings{
		AggregationMode:    store.AggregationModeFlowAggregate,
		BulkEnabled:        true,
		BulkMinHealthyLegs: 3,
	}
	plan := ResolveBulkPlan(set, []string{"a", "b"}, "a")
	if plan.Active {
		t.Fatal("expected inactive")
	}
	if plan.FallbackReason != "not_enough_legs" {
		t.Fatalf("reason=%q", plan.FallbackReason)
	}
}

func TestRewriteMatchTarget_bulk(t *testing.T) {
	in := []string{"GEOIP,ru,DIRECT", "MATCH,PROXY"}
	out := RewriteMatchTarget(in, GroupPROXYBulk)
	if !strings.Contains(strings.Join(out, "\n"), "MATCH,PROXY_BULK") {
		t.Fatalf("rules=%v", out)
	}
}

func TestBuildFromProfileBulk_generatesLoadBalance(t *testing.T) {
	p1 := store.Profile{ID: 1, Name: "n1", Type: "vless", URI: "vless://11111111-1111-1111-1111-111111111111@1.2.3.4:443?security=none"}
	p2 := store.Profile{ID: 2, Name: "n2", Type: "vless", URI: "vless://22222222-2222-2222-2222-222222222222@5.6.7.8:443?security=none"}
	set := store.Settings{
		AggregationMode:    store.AggregationModeFlowAggregate,
		BulkEnabled:        true,
		BulkMinHealthyLegs: 2,
		BulkLBStrategy:     store.BulkLBStickySessions,
	}
	yaml, _, plan, err := BuildFromProfileBulk(p1, []store.Profile{p2}, DefaultBuildOptions(), nil, "", store.RouteQuickRuDirectOnly, set)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Active {
		t.Fatalf("plan not active: %+v", plan)
	}
	if !strings.Contains(yaml, "PROXY_BULK") {
		t.Fatalf("missing PROXY_BULK:\n%s", yaml)
	}
	if !strings.Contains(yaml, "load-balance") {
		t.Fatal("missing load-balance type")
	}
	if !strings.Contains(yaml, "strategy: sticky-sessions") {
		t.Fatalf("expected sticky-sessions strategy:\n%s", yaml)
	}
	if !strings.Contains(yaml, "MATCH,PROXY_BULK") {
		t.Fatalf("MATCH not routed to bulk:\n%s", yaml)
	}
}

func TestBuildFromProfile_legacyUnchangedMatch(t *testing.T) {
	p := store.Profile{ID: 1, Name: "n1", Type: "vless", URI: "vless://11111111-1111-1111-1111-111111111111@1.2.3.4:443?security=none"}
	yaml, _, err := BuildFromProfileExtQP(p, DefaultBuildOptions(), nil, "", store.RouteQuickRuDirectOnly)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(yaml, "PROXY_BULK") {
		t.Fatal("legacy should not have PROXY_BULK")
	}
	if !strings.Contains(yaml, "MATCH,PROXY") {
		t.Fatalf("expected MATCH,PROXY:\n%s", yaml)
	}
}
