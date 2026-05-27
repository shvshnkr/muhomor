package selector

import (
	"context"
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

type fakePicker struct {
	ok     bool
	ensure error
}

func (f *fakePicker) Available() bool { return f.ok }

func (f *fakePicker) Ensure(ctx context.Context) error {
	if f.ensure != nil {
		return f.ensure
	}
	if f.ok {
		return nil
	}
	return context.Canceled
}

func (f *fakePicker) GroupDelayProfiles(ctx context.Context, profiles []store.Profile, st *store.Store) map[int64]int {
	out := make(map[int64]int, len(profiles))
	for _, p := range profiles {
		out[p.ID] = 42
	}
	return out
}

func TestTestProfilesBatchPickerPrimary(t *testing.T) {
	ctx := context.Background()
	profiles := []store.Profile{{ID: 1}, {ID: 2}, {ID: 3}}
	e := &EphemeralTester{Picker: &fakePicker{ok: true}}
	out := e.TestProfilesBatch(ctx, profiles)
	if len(out) != 3 {
		t.Fatalf("got %d want 3", len(out))
	}
	if out[1] != 42 {
		t.Fatalf("delay=%d", out[1])
	}
}
