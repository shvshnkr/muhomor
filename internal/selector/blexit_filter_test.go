package selector

import (
	"context"
	"testing"

	"github.com/muhomor/muhomor/internal/profileclass"
	"github.com/muhomor/muhomor/internal/probe/blexit"
	"github.com/muhomor/muhomor/internal/store"
)

func TestCapRankedForFallback_demotesRuExitWithoutBLOK(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ru := store.Profile{ID: 19, Name: "Russia [19]", RuExitMarked: true}
	other := store.Profile{ID: 56, Name: "Austria [56]"}
	ranked := []store.Profile{ru, other}
	tcp := map[int64]int{19: 10, 56: 50}
	got := capRankedForFallback(ctx, st, ranked, tcp, nil, true)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].ID != 56 {
		t.Fatalf("first=%d want non-ru 56", got[0].ID)
	}
	_ = profileclass.IsRuExitMarked(ru)
}

func TestApplyBLExitFilter_failClosedUsesRunner(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	_ = st.SetKV(ctx, store.KeyProbeBLExitFilterEnabled, "1")
	pool := []store.Profile{{ID: 19, Name: "Russia [19]", RuExitMarked: true}}
	urlDelays := map[int64]int{19: 100}
	s := &Selector{
		Store: st,
		Ephemeral: &EphemeralTester{
			MihomoBin: t.TempDir() + "/no-mihomo",
			Picker:    &fakePicker{ok: false, ensure: context.Canceled},
		},
	}
	_, pass, enabled := s.applyBLExitFilter(ctx, pool, urlDelays, false)
	if !enabled {
		t.Fatal("expected bl enabled fail-closed")
	}
	if !pass.FailClosed {
		t.Fatal("expected fail_closed")
	}
	if pass.Runner != "failed" {
		t.Fatalf("runner=%q want failed", pass.Runner)
	}
	suffix := blexit.ReasonSuffix(pass, true)
	if !containsAll(suffix, "bl_runner=failed", "bl_fail_closed=1") {
		t.Fatalf("suffix=%q", suffix)
	}
}

func TestRankProfiles_failClosedRuExitNotFirst(t *testing.T) {
	ru := store.Profile{ID: 19, Name: "Russia [19]", RuExitMarked: true}
	other := store.Profile{ID: 56, Name: "Austria [56]"}
	pool := []store.Profile{ru, other}
	tcp := map[int64]int{19: 10, 56: 50}
	blOK := map[int64]bool{}
	ranked := rankProfiles(pool, tcp, nil, blOK, true, nil, func(int64) bool { return false }, false, nil)
	if len(ranked) == 0 || ranked[0].ID == 19 {
		t.Fatalf("ru_exit ranked first without BL pass: %v", ranked[0].ID)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !stringsContains(s, p) {
			return false
		}
	}
	return true
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexStr(s, sub) >= 0)
}

func indexStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
