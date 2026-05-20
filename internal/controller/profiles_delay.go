package controller

import (
	"context"
	"fmt"

	"github.com/muhomor/muhomor/internal/api"
)

// TestProfileDelay runs ephemeral url-test for one stored profile.
func (r *Runtime) TestProfileDelay(ctx context.Context, profileID int64) (api.DelayTestResult, error) {
	p, err := r.Store.ProfileByID(ctx, profileID)
	if err != nil {
		return api.DelayTestResult{}, err
	}
	out := api.DelayTestResult{ProfileID: profileID}
	if r.selector == nil || r.selector.Ephemeral == nil {
		out.Error = "ephemeral tester unavailable"
		return out, nil
	}
	delay, err := r.selector.Ephemeral.TestProfile(ctx, p)
	if err != nil || delay <= 0 {
		out.Error = "url test failed"
		if err != nil {
			out.Error = err.Error()
		}
		_ = r.Store.UpdateProfileProbe(ctx, profileID, 0, 0, out.Error)
		return out, nil
	}
	out.DelayMs = delay
	_ = r.Store.UpdateProfileProbe(ctx, profileID, delay, 1, "")
	return out, nil
}

// TestGroupDelays url-tests all enabled profiles in a group and sorts by delay.
func (r *Runtime) TestGroupDelays(ctx context.Context, groupID int64) (tested, ok int, err error) {
	list, err := r.Store.ListProfilesByGroup(ctx, groupID)
	if err != nil {
		return 0, 0, err
	}
	if r.selector == nil || r.selector.Ephemeral == nil {
		return 0, 0, fmt.Errorf("ephemeral tester unavailable")
	}
	for _, p := range list {
		if !p.Enabled {
			continue
		}
		tested++
		delay, err := r.selector.Ephemeral.TestProfile(ctx, p)
		if err != nil || delay <= 0 {
			_ = r.Store.UpdateProfileProbe(ctx, p.ID, 0, 0, "url test failed")
			continue
		}
		ok++
		_ = r.Store.UpdateProfileProbe(ctx, p.ID, delay, 1, "")
	}
	if err := r.Store.SortGroupProfilesByDelay(ctx, groupID); err != nil {
		return tested, ok, err
	}
	return tested, ok, nil
}
