package aggregate

import (
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestScheduler_bulkWeightsSum(t *testing.T) {
	sch := Scheduler{Config: ConfigForPreset(PresetNormal)}
	profiles := []store.Profile{{ID: 1}, {ID: 2}, {ID: 3}}
	inputs := []ChannelInput{
		{ProfileID: 1, RTTMs: 100, GoodputKbps: 20000},
		{ProfileID: 2, RTTMs: 150, GoodputKbps: 15000},
		{ProfileID: 3, RTTMs: 200, GoodputKbps: 10000},
	}
	res := sch.Schedule(profiles, inputs, FlowBulk, WLPolicy{MaxPct: 25})
	sum := 0
	for _, w := range res.BulkWeights {
		sum += w
	}
	if sum < 900 || sum > 1100 {
		t.Fatalf("weights sum %d", sum)
	}
}

func TestScheduler_singleStreamConservative(t *testing.T) {
	sch := Scheduler{Config: Config{MaxChannels: 4}}
	profiles := []store.Profile{{ID: 10}, {ID: 20}}
	inputs := []ChannelInput{
		{ProfileID: 10, RTTMs: 50, GoodputKbps: 5000},
		{ProfileID: 20, RTTMs: 40, GoodputKbps: 50000},
	}
	res := sch.Schedule(profiles, inputs, FlowSingleStream, WLPolicy{})
	if res.PrimaryID != 20 {
		t.Fatalf("primary %d", res.PrimaryID)
	}
	if res.Reason != "multipath:single_conservative" {
		t.Fatalf("reason %q", res.Reason)
	}
}
