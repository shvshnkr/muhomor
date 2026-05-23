package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestAggregationDefaults_emptyKV(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	if got := st.AggregationMode(ctx); got != AggregationModeLegacy {
		t.Fatalf("AggregationMode=%q want legacy", got)
	}
	if st.BulkEnabled(ctx) {
		t.Fatal("BulkEnabled want false")
	}
}
