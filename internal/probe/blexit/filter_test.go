package blexit

import (
	"strings"
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestMergeSuite_OR(t *testing.T) {
	perURL := []map[int64]int{
		{1: 100, 2: 0},
		{1: 0, 2: 200},
	}
	pass := MergeSuite(perURL, 1)
	if !pass[1] || !pass[2] {
		t.Fatalf("OR merge: %+v", pass)
	}
}

func TestApplyFilter_scopeAll(t *testing.T) {
	url := map[int64]int{10: 50, 20: 60, 30: 70}
	pool := []store.Profile{
		{ID: 10, Name: "Russia [01]", RuExitMarked: true},
		{ID: 20, Name: "DE"},
		{ID: 30, Name: "NL"},
	}
	pass := PassResult{
		PerProfile:  map[int64]bool{20: true, 30: true},
		TestedCount: 3,
		PassCount:   2,
		Scope:       ScopeAll,
	}
	out := ApplyFilter(url, pool, pass, ScopeAll)
	if len(out) != 2 || out[20] != 60 || out[30] != 70 {
		t.Fatalf("expected BL pass only: %+v", out)
	}
}

func TestApplyFilter_scopeRuExit(t *testing.T) {
	url := map[int64]int{10: 50, 20: 60}
	pool := []store.Profile{
		{ID: 10, Name: "Russia [01]", RuExitMarked: true},
		{ID: 20, Name: "DE"},
	}
	pass := PassResult{
		PerProfile:  map[int64]bool{20: true},
		TestedCount: 1,
		Scope:       ScopeRuExit,
	}
	out := ApplyFilter(url, pool, pass, ScopeRuExit)
	if _, ok := out[10]; ok {
		t.Fatal("ru_exit without BL should be stripped")
	}
	if out[20] != 60 {
		t.Fatal("non-ru should remain")
	}
}

func TestPickBLModeCandidates(t *testing.T) {
	pool := []store.Profile{
		{ID: 1, Name: "Anycast [BL]"},
		{ID: 2, Name: "DE"},
		{ID: 3, Name: "Russia [01]", RuExitMarked: true},
	}
	url := map[int64]int{1: 50, 2: 40, 3: 30}
	got := PickBLModeCandidates(pool, url, 0)
	if len(got) != 2 {
		t.Fatalf("got len=%d want 2", len(got))
	}
	if got[0].ID != 3 || got[1].ID != 1 {
		t.Fatalf("order=%d,%d want 3,1 by delay", got[0].ID, got[1].ID)
	}
}

func TestApplyFilter_scopeRuExit_stripsBLTag(t *testing.T) {
	url := map[int64]int{10: 50, 11: 60}
	pool := []store.Profile{
		{ID: 10, Name: "Anycast [BL]"},
		{ID: 11, Name: "DE"},
	}
	pass := PassResult{
		PerProfile:  map[int64]bool{11: true},
		TestedCount: 1,
		Scope:       ScopeRuExit,
	}
	out := ApplyFilter(url, pool, pass, ScopeRuExit)
	if _, ok := out[10]; ok {
		t.Fatal("[bl] without BL pass should be stripped")
	}
	if out[11] != 60 {
		t.Fatal("non-uplink should remain")
	}
}

func TestPickRuExitCandidates(t *testing.T) {
	pool := []store.Profile{
		{ID: 1, Name: "Russia [01]", RuExitMarked: true},
		{ID: 2, Name: "DE"},
	}
	url := map[int64]int{1: 50, 2: 40}
	got := PickRuExitCandidates(pool, url, 0)
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("got %+v", got)
	}
}

func TestReasonSuffix_allFailedOnlyAllScope(t *testing.T) {
	all := ReasonSuffix(PassResult{TestedCount: 5, RejectCount: 5, Scope: ScopeAll}, true)
	if !strings.Contains(all, "bl_all_failed=1") {
		t.Fatalf("all scope: %q", all)
	}
	ru := ReasonSuffix(PassResult{TestedCount: 5, RejectCount: 5, Scope: ScopeRuExit}, true)
	if strings.Contains(ru, "bl_all_failed=1") {
		t.Fatalf("ru scope should not all-fail: %q", ru)
	}
}
