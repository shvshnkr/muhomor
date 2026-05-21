package aggregate

import (
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestComputeScore_narrowPipeWorseThanHealthy(t *testing.T) {
	lowRTT := ComputeScore(ChannelInput{ProfileID: 1, RTTMs: 80, GoodputKbps: 3000, State: store.ChannelCongested})
	highGood := ComputeScore(ChannelInput{ProfileID: 2, RTTMs: 120, GoodputKbps: 40000, State: store.ChannelHealthy})
	if lowRTT.Score <= highGood.Score {
		t.Fatalf("congested narrow should score worse: %d vs %d", lowRTT.Score, highGood.Score)
	}
	if lowRTT.Reason != "narrow_pipe" && lowRTT.Reason != "congested" {
		t.Fatalf("reason %q", lowRTT.Reason)
	}
}

func TestUpdateStateFromSample_hysteresis(t *testing.T) {
	st := store.ChannelHealthy
	st = UpdateStateFromSample(st, 100, 10, 5, true)
	if st != store.ChannelCongested {
		t.Fatalf("got %d", st)
	}
	st = UpdateStateFromSample(st, 100, 10, 5, true)
	if st != store.ChannelDegraded {
		t.Fatalf("got %d", st)
	}
}

func TestGoodputFromDelayKbps(t *testing.T) {
	if g := GoodputFromDelayKbps(100); g < 50 {
		t.Fatalf("g %d", g)
	}
}
