package store

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// openBenchDB opens an isolated DB for benchmarks and tests.

// seedBenchProfiles inserts n synthetic vless profiles with mixed probe states.
func seedBenchProfiles(ctx context.Context, st *Store, n int) error {
	now := time.Now().UTC().Format(time.RFC3339)
	states := []int{ProbeAlive, ProbeCandidate, ProbeSuspect, ProbeDead, ProbeUnknown}
	for i := 0; i < n; i++ {
		uri := fmt.Sprintf("vless://u@%d.0.0.%d:443?security=none#b%d", i/254, i%254, i)
		id, err := st.UpsertProfile(ctx, fmt.Sprintf("bench-%d", i), "vless", uri)
		if err != nil {
			return err
		}
		pstate := states[i%len(states)]
		ewma := 50 + (i % 500)
		_, err = st.db.ExecContext(ctx, `
			UPDATE profiles SET
			  probe_state = ?,
			  probe_ewma_delay_ms = ?,
			  probe_last_ok_at = ?,
			  probe_last_checked_at = ?,
			  probe_next_probe_at = ?
			WHERE id = ?`,
			pstate, ewma, now, now, now, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func BenchmarkListWarmAlive2K(b *testing.B) {
	st := openBenchDB(b)
	defer st.Close()
	ctx := context.Background()
	if err := seedBenchProfiles(ctx, st, 2000); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := st.ListWarmAliveProfiles(ctx, 20*time.Minute, 64); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkListDueProbe2K(b *testing.B) {
	st := openBenchDB(b)
	defer st.Close()
	ctx := context.Background()
	if err := seedBenchProfiles(ctx, st, 2000); err != nil {
		b.Fatal(err)
	}
	now := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := st.ListProfilesDueProbe(ctx, 64, now); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCountProbeStates2K(b *testing.B) {
	st := openBenchDB(b)
	defer st.Close()
	ctx := context.Background()
	if err := seedBenchProfiles(ctx, st, 2000); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := st.CountProbeStates(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func openBenchDB(tb testing.TB) *Store {
	tb.Helper()
	st, err := Open(filepath.Join(tb.TempDir(), "bench.db"))
	if err != nil {
		tb.Fatal(err)
	}
	return st
}

// TestWarmAliveFreshness2K ensures stale checked_at rows are excluded at 2k scale.
func TestWarmAliveFreshness2K(t *testing.T) {
	st := openBenchDB(t)
	defer st.Close()
	ctx := context.Background()
	if err := seedBenchProfiles(ctx, st, 200); err != nil {
		t.Fatal(err)
	}
	old := timeToDB(time.Now().Add(-48 * time.Hour))
	_, err := st.db.ExecContext(ctx, `UPDATE profiles SET probe_last_checked_at = ? WHERE id % 5 = 0`, old)
	if err != nil {
		t.Fatal(err)
	}
	warm, err := st.ListWarmAliveProfiles(ctx, 20*time.Minute, 64)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range warm {
		m, err := st.ProbeMetaByID(ctx, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !m.IsWarmAlive(20 * time.Minute) {
			t.Fatalf("profile %d not warm", p.ID)
		}
	}
}
