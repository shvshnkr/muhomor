package aggregate

import (
	"sort"

	"github.com/muhomor/muhomor/internal/store"
)

// Scheduler picks channels and bulk weights for desktop multipath.
type Scheduler struct {
	Config Config
}

// ScheduleResult output of one scheduling pass.
type ScheduleResult struct {
	PrimaryID   int64
	ChannelPool []int64
	BulkWeights map[int64]int // permille sum ~1000
	Reason      string
	Scores      []ChannelScore
}

// Schedule ranks profiles and builds channel pool + weights.
func (sch *Scheduler) Schedule(profiles []store.Profile, inputs []ChannelInput, flow FlowClass, pol WLPolicy) ScheduleResult {
	scores := make([]ChannelScore, 0, len(inputs))
	inByID := map[int64]ChannelInput{}
	for _, in := range inputs {
		inByID[in.ProfileID] = in
	}
	for _, p := range profiles {
		in, ok := inByID[p.ID]
		if !ok {
			in = ChannelInput{ProfileID: p.ID, WLBuiltin: p.WLBuiltinPool}
		}
		scores = append(scores, ComputeScore(in))
	}
	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].Score != scores[j].Score {
			return scores[i].Score < scores[j].Score
		}
		return scores[i].ProfileID < scores[j].ProfileID
	})
	byID := map[int64]store.Profile{}
	for _, p := range profiles {
		byID[p.ID] = p
	}
	var ordered []store.Profile
	for _, sc := range scores {
		if p, ok := byID[sc.ProfileID]; ok && sc.Score < 1<<28 {
			ordered = append(ordered, p)
		}
	}
	filtered := FilterChannelPool(ordered, WLPolicy{
		EmergencyOnly: pol.EmergencyOnly,
		MaxPct:        pol.MaxPct,
		SubsDegraded:  SubsPoolDegraded(scores, profiles),
	})
	maxCh := sch.Config.MaxChannels
	if maxCh <= 0 {
		maxCh = 6
	}
	if len(filtered) > maxCh {
		filtered = filtered[:maxCh]
	}
	pool := make([]int64, len(filtered))
	for i, p := range filtered {
		pool[i] = p.ID
	}
	var primary int64
	reason := "multipath:interactive"
	if len(pool) > 0 {
		primary = pool[0]
	}
	weights := sch.bulkWeights(filtered, scores, flow)
	switch flow {
	case FlowBulk:
		reason = "multipath:bulk_weighted"
	case FlowBackground:
		reason = "multipath:background_spare"
	case FlowSingleStream:
		reason = "multipath:single_conservative"
		if len(pool) > 0 {
			primary = pickStableLowest(scores, pool)
		}
	}
	return ScheduleResult{
		PrimaryID:   primary,
		ChannelPool: pool,
		BulkWeights: weights,
		Reason:      reason,
		Scores:      scores,
	}
}

func pickStableLowest(scores []ChannelScore, pool []int64) int64 {
	inPool := map[int64]struct{}{}
	for _, id := range pool {
		inPool[id] = struct{}{}
	}
	best := pool[0]
	bestScore := 1 << 30
	for _, sc := range scores {
		if _, ok := inPool[sc.ProfileID]; !ok {
			continue
		}
		if sc.Score < bestScore {
			bestScore = sc.Score
			best = sc.ProfileID
		}
	}
	return best
}

func (sch *Scheduler) bulkWeights(filtered []store.Profile, scores []ChannelScore, flow FlowClass) map[int64]int {
	out := map[int64]int{}
	if flow != FlowBulk && flow != FlowBackground {
		if len(filtered) > 0 {
			out[filtered[0].ID] = 1000
		}
		return out
	}
	scByID := map[int64]int{}
	for _, sc := range scores {
		scByID[sc.ProfileID] = sc.Score
	}
	cap := sch.Config.BulkWeightCap
	if cap <= 0 {
		cap = 400
	}
	var invSum int
	inv := make([]int, len(filtered))
	for i, p := range filtered {
		sc := scByID[p.ID]
		if sc <= 0 {
			sc = 1
		}
		inv[i] = 100000 / sc
		invSum += inv[i]
	}
	if invSum == 0 {
		return out
	}
	assigned := 0
	for i, p := range filtered {
		w := inv[i] * 1000 / invSum
		if w > cap {
			w = cap
		}
		out[p.ID] = w
		assigned += w
	}
	if len(filtered) > 0 && assigned < 1000 {
		out[filtered[0].ID] += 1000 - assigned
	}
	return out
}
