package selector

import (
	"context"
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestRegression_TestProfilesBatch_fallsBackWithoutPicker(t *testing.T) {
	ctx := context.Background()
	profiles := []store.Profile{{ID: 1}}
	e := &EphemeralTester{}
	out := e.TestProfilesBatch(ctx, profiles)
	if len(out) != 0 {
		t.Fatalf("without picker expected empty map, got %d entries", len(out))
	}
}

func TestRegression_PickerReady_mihomoTypeUsesEnsure(t *testing.T) {
	ctx := context.Background()
	fp := &fakePicker{ok: false, ensure: context.Canceled}
	if pickerReady(ctx, fp) {
		t.Fatal("Ensure failure must make pickerReady false")
	}
	fp.ok = true
	fp.ensure = nil
	if !pickerReady(ctx, fp) {
		t.Fatal("Ensure success must make pickerReady true")
	}
}
