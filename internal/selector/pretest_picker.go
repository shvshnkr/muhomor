package selector

import (
	"context"
	"fmt"

	"github.com/muhomor/muhomor/internal/standby"
	"github.com/muhomor/muhomor/internal/store"
)

const urlTestCapPicker = 12

// TestProfilesBatch URL-tests via picker batch when available; otherwise parallel+micro fallback.
func (e *EphemeralTester) testProfilesPickerPrimary(ctx context.Context, profiles []store.Profile) (map[int64]int, bool) {
	if e.Picker == nil || len(profiles) == 0 || !pickerReady(ctx, e.Picker) {
		return nil, false
	}
	if len(profiles) > urlTestCapPicker {
		profiles = profiles[:urlTestCapPicker]
	}
	if e.Activity != nil {
		e.Activity(ctx, fmt.Sprintf("Picker batch %d…", len(profiles)))
	}
	out := e.Picker.GroupDelayProfiles(ctx, profiles, e.Store)
	if len(out) == 0 {
		return nil, false
	}
	if e.Log != nil {
		e.Log.Info("picker pretest", "ok", len(out), "candidates", len(profiles), "event", "H4-picker")
	}
	if e.Activity != nil {
		e.Activity(ctx, fmt.Sprintf("Picker готов: ok=%d", len(out)))
	}
	return out, true
}

func (e *EphemeralTester) testProfilesPickerWithCatchup(ctx context.Context, sorted []store.Profile) map[int64]int {
	out, ok := e.testProfilesPickerPrimary(ctx, sorted)
	if !ok {
		return nil
	}
	if len(out) >= standby.HotSize {
		return out
	}
	if len(sorted) <= urlTestCapPicker {
		return out
	}
	end := urlTestCapPicker * 2
	if end > len(sorted) {
		end = len(sorted)
	}
	batch2 := sorted[urlTestCapPicker:end]
	if e.Activity != nil {
		e.Activity(ctx, fmt.Sprintf("Picker batch %d (догон)…", len(batch2)))
	}
	part := e.Picker.GroupDelayProfiles(ctx, batch2, e.Store)
	for id, ms := range part {
		if ms > 0 {
			out[id] = ms
		}
	}
	return out
}
