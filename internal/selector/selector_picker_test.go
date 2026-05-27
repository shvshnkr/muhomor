package selector

import (
	"context"
	"testing"
)

func TestColdPrepareProbes_usesPickerWhenEnsureOk(t *testing.T) {
	ctx := context.Background()
	s := &Selector{
		Ephemeral: &EphemeralTester{
			Picker: &fakePicker{ok: true},
		},
	}
	if !pickerReady(ctx, s.Ephemeral.Picker) {
		t.Fatal("pickerReady should succeed when Ensure ok")
	}
}

func TestColdPrepareProbes_skipsPickerWhenEnsureFails(t *testing.T) {
	ctx := context.Background()
	s := &Selector{
		Ephemeral: &EphemeralTester{
			Picker: &fakePicker{ok: false, ensure: context.Canceled},
		},
	}
	if pickerReady(ctx, s.Ephemeral.Picker) {
		t.Fatal("pickerReady should fail when Ensure fails even if Available false")
	}
}
