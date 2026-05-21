package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMigrate_v5_probeColumns(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	id, err := st.UpsertProfile(ctx, "p1", "vless", "vless://u@1.2.3.4:443?security=none#t")
	if err != nil {
		t.Fatal(err)
	}
	_ = st.UpdateProfileProbe(ctx, id, 120, StatusAvailable, "")
	m, err := st.ProbeMetaByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if m.State != ProbeAlive {
		t.Fatalf("state %d", m.State)
	}
	if m.EWMADelayMs != 120 {
		t.Fatalf("ewma %d", m.EWMADelayMs)
	}
}
