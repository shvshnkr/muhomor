package store

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestProbeStats_JSON_roundtrip(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	in := ProbeStats{
		TotalEnabled: 2000, Alive: 120, Candidate: 80,
		LastTickChecked: 64, LastTickOK: 40, LastTickFail: 24,
		SchedulerEnabled: true, Preset: "normal",
		LastSelectReason: "warm:best=1 queue=48",
	}
	if err := st.SetProbeStats(ctx, in); err != nil {
		t.Fatal(err)
	}
	out := st.GetProbeStats(ctx)
	if out.TotalEnabled != 2000 || out.Alive != 120 {
		t.Fatalf("got %+v", out)
	}
	if out.LastSelectReason != in.LastSelectReason {
		t.Fatalf("reason %q", out.LastSelectReason)
	}
	raw, _ := st.GetKV(ctx, KeyProbeSchedulerStats)
	var parsed ProbeStats
	if json.Unmarshal([]byte(raw), &parsed) != nil {
		t.Fatal("not json")
	}
}

func TestParseLegacyProbeStats(t *testing.T) {
	base := ProbeStats{TotalEnabled: 548}
	out := parseLegacyProbeStats("checked=64 ok=40 fail=24 alive=120 dead=300 cemetery=50", base)
	if out.LastTickChecked != 64 || out.Alive != 120 {
		t.Fatalf("%+v", out)
	}
}
