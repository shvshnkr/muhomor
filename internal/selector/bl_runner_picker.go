package selector

import (
	"context"

	"github.com/muhomor/muhomor/internal/mihomo"
	"github.com/muhomor/muhomor/internal/probe/blexit"
	"github.com/muhomor/muhomor/internal/store"
)

// PickerSuiteRunner runs BL suite via long-lived picker mihomo.
type PickerSuiteRunner struct {
	Picker *mihomo.Picker
}

func (r *PickerSuiteRunner) Name() string { return "picker" }

func (r *PickerSuiteRunner) RunSuite(ctx context.Context, profiles []store.Profile, st *store.Store, urls []string, minPass int) (map[int64]bool, error) {
	if r.Picker == nil {
		return nil, nil
	}
	if err := r.Picker.Ensure(ctx); err != nil {
		return nil, err
	}
	out := r.Picker.GroupDelayProfilesBLSuite(ctx, profiles, st, urls, minPass)
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

var _ blexit.SuiteRunner = (*PickerSuiteRunner)(nil)
