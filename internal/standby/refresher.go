package standby

import (
	"context"
	"log/slog"
	"time"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/probe/blexit"
	"github.com/muhomor/muhomor/internal/store"
)

// PickerBatch runs batch URL-test on profiles (picker or ephemeral fallback).
type PickerBatch interface {
	Available() bool
	Ensure(ctx context.Context) error
	GroupDelayProfiles(ctx context.Context, profiles []store.Profile, st *store.Store) map[int64]int
}

// Refresher maintains hot/warm standby pools in the background.
type Refresher struct {
	Store         *store.Store
	Picker        PickerBatch
	BLPrimary     blexit.SuiteRunner
	BLFallback    blexit.SuiteRunner
	Log           *slog.Logger
	Activity      func(context.Context, string)
	WLOnly        func(context.Context) bool
	IsConnected   func(context.Context) bool
	ProdDelay     func(context.Context, string) (int, error)
	ActiveSession func(context.Context) (int64, string, bool)
}

// Run starts hot (60s) and warm (3m) refresh loops until ctx is done.
func (r *Refresher) Run(ctx context.Context) {
	hotTick := time.NewTicker(HotRefreshInterval)
	warmTick := time.NewTicker(WarmRefreshInterval)
	defer hotTick.Stop()
	defer warmTick.Stop()
	r.refreshHot(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-hotTick.C:
			r.refreshHot(ctx)
		case <-warmTick.C:
			r.refreshWarm(ctx)
		}
	}
}

// RefreshUrgent forces a picker batch on up to n candidates (session recover).
func (r *Refresher) RefreshUrgent(ctx context.Context, n int) {
	if n <= 0 {
		n = CandidateBatchCap
	}
	wl := false
	if r.WLOnly != nil {
		wl = r.WLOnly(ctx)
	}
	cands, err := TopCandidates(ctx, r.Store, n, wl)
	if err != nil || len(cands) == 0 {
		return
	}
	if r.Activity != nil {
		r.Activity(ctx, "Срочное обновление standby…")
	}
	r.runBatch(ctx, cands)
}

func (r *Refresher) refreshHot(ctx context.Context) {
	if IsHotFresh(ctx, r.Store) {
		return
	}
	wl := false
	if r.WLOnly != nil {
		wl = r.WLOnly(ctx)
	}
	cands, err := TopCandidates(ctx, r.Store, CandidateBatchCap, wl)
	if err != nil || len(cands) == 0 {
		return
	}
	if r.Activity != nil {
		r.Activity(ctx, "Обновление standby…")
	}
	r.runBatch(ctx, cands)
}

func (r *Refresher) refreshWarm(ctx context.Context) {
	wl := false
	if r.WLOnly != nil {
		wl = r.WLOnly(ctx)
	}
	cands, err := TopCandidates(ctx, r.Store, CandidateBatchCap, wl)
	if err != nil || len(cands) == 0 {
		return
	}
	r.runBatch(ctx, cands)
}

func (r *Refresher) refreshActiveProduction(ctx context.Context) {
	if r.Store == nil || r.ProdDelay == nil || r.ActiveSession == nil {
		return
	}
	if r.IsConnected != nil && !r.IsConnected(ctx) {
		return
	}
	pid, proxy, ok := r.ActiveSession(ctx)
	if !ok || pid <= 0 || proxy == "" {
		return
	}
	delay, err := r.ProdDelay(ctx, proxy)
	if err != nil || delay <= 0 {
		return
	}
	PromoteAfterConnect(ctx, r.Store, pid, proxy, delay)
}

func (r *Refresher) annotateBLDelays(ctx context.Context, profiles []store.Profile, gstatic map[int64]int, ranked []Entry, wlOnly bool) {
	if wlOnly || r.Store == nil || !r.Store.BLExitFilterEnabled(ctx) {
		return
	}
	var candidates []store.Profile
	if r.Store.BLExitFilterScope(ctx) == store.BLExitScopeAll {
		candidates = blexit.PickBLCandidates(profiles, gstatic, 0)
	} else {
		candidates = blexit.PickRuExitCandidates(profiles, gstatic, 0)
	}
	if len(candidates) == 0 {
		return
	}
	runners := r.blSuiteRunners()
	if len(runners) == 0 {
		return
	}
	urls := r.Store.BLExitTestURLs(ctx)
	minPass := r.Store.BLExitMinPass(ctx)
	blPass, _, _ := blexit.RunSuiteBatches(ctx, runners, candidates, r.Store, urls, minPass)
	for i := range ranked {
		if blPass[ranked[i].ProfileID] {
			if ms := gstatic[ranked[i].ProfileID]; ms > 0 {
				ranked[i].BLDelayMs = ms
				_ = r.Store.SetBLExitOK(ctx, ranked[i].ProfileID, ms)
			}
		}
	}
}

func (r *Refresher) blSuiteRunners() []blexit.SuiteRunner {
	var out []blexit.SuiteRunner
	if r.BLPrimary != nil {
		out = append(out, r.BLPrimary)
	}
	if r.BLFallback != nil {
		out = append(out, r.BLFallback)
	}
	return out
}

func (r *Refresher) runBatch(ctx context.Context, profiles []store.Profile) {
	r.refreshActiveProduction(ctx)
	if r.Picker == nil {
		return
	}
	if err := r.Picker.Ensure(ctx); err != nil {
		return
	}
	wl := false
	if r.WLOnly != nil {
		wl = r.WLOnly(ctx)
	}
	delays := r.Picker.GroupDelayProfiles(ctx, profiles, r.Store)
	if len(delays) == 0 {
		return
	}
	var ranked []Entry
	now := time.Now().UTC()
	for _, p := range profiles {
		ms, ok := delays[p.ID]
		if !ok || ms <= 0 {
			continue
		}
		name, _ := configgen.ProxyNameForProfile(p)
		ranked = append(ranked, Entry{
			ProfileID:  p.ID,
			ProxyName:  name,
			DelayMs:    ms,
			VerifiedAt: now,
		})
	}
	r.annotateBLDelays(ctx, profiles, delays, ranked, wl)
	ApplyDelayResults(ctx, r.Store, ranked)
	if r.Log != nil {
		r.Log.Info("standby refreshed", "ok", len(ranked), "event", "H4-standby")
	}
}
