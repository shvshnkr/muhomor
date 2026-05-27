package blexit

import (
	"fmt"

	"github.com/muhomor/muhomor/internal/profileclass"
	"github.com/muhomor/muhomor/internal/store"
)

const TestCap = 12

const (
	ScopeRuExit = "ru_exit"
	ScopeAll    = "all"
)

// PassResult is BL-over-proxy outcome for pre-connect filtering.
type PassResult struct {
	PerProfile   map[int64]bool
	PassCount    int
	RejectCount  int
	TestedCount  int
	Scope        string
	Runner       string // picker | ephemeral | failed | none
	FailClosed   bool
}

// MergeSuite OR-merges per-URL delay maps (minPass URLs must pass).
func MergeSuite(perURL []map[int64]int, minPass int) map[int64]bool {
	if minPass <= 0 {
		minPass = 1
	}
	counts := map[int64]int{}
	for _, m := range perURL {
		for id, ms := range m {
			if ms > 0 {
				counts[id]++
			}
		}
	}
	out := make(map[int64]bool, len(counts))
	for id, n := range counts {
		if n >= minPass {
			out[id] = true
		}
	}
	return out
}

// PickBLCandidates returns profiles with live gstatic/url delay (sorted fastest first).
func PickBLCandidates(pool []store.Profile, urlDelays map[int64]int, cap int) []store.Profile {
	var cand []store.Profile
	for _, p := range pool {
		if urlDelays[p.ID] > 0 {
			cand = append(cand, p)
		}
	}
	sortByURLDelay(cand, urlDelays)
	if cap > 0 && len(cand) > cap {
		cand = cand[:cap]
	}
	return cand
}

// PickBLModeCandidates returns BL ISP uplink profiles (ru_exit or [bl] tag) with gstatic ok.
func PickBLModeCandidates(pool []store.Profile, urlDelays map[int64]int, cap int) []store.Profile {
	var cand []store.Profile
	for _, p := range pool {
		if !profileclass.IsBLModeUplink(p) {
			continue
		}
		if urlDelays[p.ID] > 0 {
			cand = append(cand, p)
		}
	}
	sortByURLDelay(cand, urlDelays)
	if cap > 0 && len(cand) > cap {
		cand = cand[:cap]
	}
	return cand
}

// PickRuExitCandidates returns ru_exit profiles with gstatic ok.
func PickRuExitCandidates(pool []store.Profile, urlDelays map[int64]int, cap int) []store.Profile {
	var cand []store.Profile
	for _, p := range pool {
		if !profileclass.IsRuExitMarked(p) {
			continue
		}
		if urlDelays[p.ID] > 0 {
			cand = append(cand, p)
		}
	}
	sortByURLDelay(cand, urlDelays)
	if cap > 0 && len(cand) > cap {
		cand = cand[:cap]
	}
	return cand
}

func sortByURLDelay(profiles []store.Profile, urlDelays map[int64]int) {
	for i := 0; i < len(profiles); i++ {
		for j := i + 1; j < len(profiles); j++ {
			if urlDelays[profiles[j].ID] < urlDelays[profiles[i].ID] {
				profiles[i], profiles[j] = profiles[j], profiles[i]
			}
		}
	}
}

// ApplyFilter applies BL pass results; scope ru_exit strips failed BL uplink (ru_exit or [bl]), scope all keeps BL pass only.
func ApplyFilter(urlDelays map[int64]int, pool []store.Profile, pass PassResult, scope string) map[int64]int {
	if len(urlDelays) == 0 {
		return urlDelays
	}
	if scope == ScopeAll {
		out := make(map[int64]int, len(pass.PerProfile))
		for id, ms := range urlDelays {
			if pass.PerProfile[id] {
				out[id] = ms
			}
		}
		return out
	}
	out := make(map[int64]int, len(urlDelays))
	for id, ms := range urlDelays {
		out[id] = ms
	}
	for _, p := range pool {
		if !profileclass.IsBLModeUplink(p) {
			continue
		}
		if pass.PerProfile[p.ID] {
			continue
		}
		delete(out, p.ID)
	}
	return out
}

// BuildPassResult builds PassResult from merged BL pass map.
func BuildPassResult(perProfile map[int64]bool, testedCount int, scope string) PassResult {
	if perProfile == nil {
		perProfile = map[int64]bool{}
	}
	if scope == "" {
		scope = ScopeRuExit
	}
	res := PassResult{
		PerProfile:  perProfile,
		TestedCount: testedCount,
		Scope:       scope,
	}
	for _, ok := range perProfile {
		if ok {
			res.PassCount++
		}
	}
	res.RejectCount = testedCount - res.PassCount
	if res.RejectCount < 0 {
		res.RejectCount = 0
	}
	return res
}

// ReasonSuffix appends BL filter telemetry to select reason.
func ReasonSuffix(pass PassResult, enabled bool) string {
	if !enabled {
		return ""
	}
	scope := pass.Scope
	if scope == "" {
		scope = ScopeRuExit
	}
	runner := pass.Runner
	if runner == "" {
		runner = "none"
	}
	fail := ""
	if pass.FailClosed {
		fail = " bl_fail_closed=1"
	}
	if pass.TestedCount > 0 && pass.PassCount == 0 && scope == ScopeAll {
		return fmt.Sprintf(" bl_ok=0 bl_rej=%d bl_filter_enabled=1 bl_scope=%s bl_runner=%s%s bl_all_failed=1", pass.RejectCount, scope, runner, fail)
	}
	return fmt.Sprintf(" bl_ok=%d bl_rej=%d bl_filter_enabled=1 bl_scope=%s bl_runner=%s%s", pass.PassCount, pass.RejectCount, scope, runner, fail)
}
