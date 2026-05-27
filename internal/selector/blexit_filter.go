package selector

import (
	"context"
	"fmt"

	"github.com/muhomor/muhomor/internal/mihomo"
	"github.com/muhomor/muhomor/internal/probe/blexit"
	"github.com/muhomor/muhomor/internal/store"
)

func (s *Selector) applyBLExitFilter(ctx context.Context, pool []store.Profile, urlDelays map[int64]int, wlOnly bool) (map[int64]int, blexit.PassResult, bool) {
	pass := blexit.PassResult{PerProfile: map[int64]bool{}}
	if wlOnly || s.Store == nil || !s.Store.BLExitFilterEnabled(ctx) {
		return urlDelays, pass, false
	}
	if len(urlDelays) == 0 {
		return urlDelays, pass, false
	}
	scope := s.Store.BLExitFilterScope(ctx)
	var candidates []store.Profile
	if scope == blexit.ScopeAll {
		candidates = blexit.PickBLCandidates(pool, urlDelays, 0)
	} else {
		candidates = blexit.PickBLModeCandidates(pool, urlDelays, 0)
	}
	if len(candidates) == 0 {
		return urlDelays, pass, false
	}
	urls := s.Store.BLExitTestURLs(ctx)
	minPass := s.Store.BLExitMinPass(ctx)
	if s.Activity != nil {
		s.Activity(ctx, fmt.Sprintf("BL (%s): проверка %d…", scope, len(candidates)))
	}

	primary, fallback := s.blSuiteRunners()
	runners := []blexit.SuiteRunner{}
	if primary != nil {
		runners = append(runners, primary)
	}
	if fallback != nil {
		runners = append(runners, fallback)
	}
	perProfile, runnerName, suiteRan := blexit.RunSuiteBatches(ctx, runners, candidates, s.Store, urls, minPass)

	pass = blexit.BuildPassResult(perProfile, len(candidates), scope)
	pass.Runner = runnerName
	if !suiteRan {
		pass.FailClosed = true
		pass.Runner = "failed"
	}

	filtered := urlDelays
	if suiteRan {
		filtered = blexit.ApplyFilter(urlDelays, pool, pass, scope)
	}
	for id, ok := range perProfile {
		if !ok {
			continue
		}
		ms := filtered[id]
		if ms <= 0 {
			ms = urlDelays[id]
		}
		if ms > 0 {
			_ = s.Store.SetBLExitOK(ctx, id, ms)
		}
	}
	if s.Log != nil {
		s.Log.Info("BL-exit-filter",
			"bl_ok", pass.PassCount,
			"bl_rej", pass.RejectCount,
			"tested", pass.TestedCount,
			"scope", scope,
			"runner", pass.Runner,
			"fail_closed", pass.FailClosed,
			"event", "BL-exit-filter")
	}
	blEnabled := suiteRan || pass.FailClosed
	return filtered, pass, blEnabled
}

func (s *Selector) blSuiteRunners() (primary, fallback blexit.SuiteRunner) {
	bin := mihomo.ResolveBin()
	if s.Ephemeral != nil {
		bin = s.Ephemeral.mihomoBinFast()
	}
	if s.Ephemeral != nil {
		if p, ok := s.Ephemeral.Picker.(*mihomo.Picker); ok && p != nil {
			primary = &PickerSuiteRunner{Picker: p}
		}
	}
	fallback = &EphemeralSuiteRunner{MihomoBin: bin}
	return primary, fallback
}

func blOkMap(pass blexit.PassResult, enabled bool) map[int64]bool {
	if !enabled {
		return nil
	}
	if pass.FailClosed && len(pass.PerProfile) == 0 {
		return map[int64]bool{}
	}
	return pass.PerProfile
}

// blUplinkOnlyScope: when filter ran in ru_exit/bl_mode scope, ranking penalizes failed uplink nodes only.
func blUplinkOnlyScope(pass blexit.PassResult, enabled bool) bool {
	if !enabled {
		return false
	}
	return pass.Scope != blexit.ScopeAll
}
