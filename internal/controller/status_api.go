package controller

import (
	"context"

	"github.com/muhomor/muhomor/internal/api"
	"github.com/muhomor/muhomor/internal/store"
)

func (r *Runtime) statusSnapshot(ctx context.Context) api.ServiceStatus {
	st := r.Status()
	activity, _ := r.Store.GetKV(ctx, store.KeySimpleModeActivity)
	out := api.ServiceStatus{
		State:              api.ServiceState(st.State),
		Connected:          st.ConnectedBool(),
		ProfileID:          st.ProfileID,
		ProfileName:        st.ProfileName,
		ProxyName:          st.ProxyName,
		SubscriptionSource: st.SubscriptionSource,
		ActivityText:       activity,
	}
	if r.Store != nil {
		ps := r.Store.GetProbeStats(ctx)
		out.Probe = &api.ProbeProgress{
			TotalEnabled:      ps.TotalEnabled,
			Alive:             ps.Alive,
			Candidate:         ps.Candidate,
			Suspect:           ps.Suspect,
			Dead:              ps.Dead,
			Cemetery:          ps.Cemetery,
			LastTickChecked:   ps.LastTickChecked,
			LastTickOK:        ps.LastTickOK,
			LastTickFail:      ps.LastTickFail,
			SchedulerEnabled:  ps.SchedulerEnabled,
			WarmSelectEnabled: ps.WarmSelectEnabled,
			Preset:            ps.Preset,
			LastSelectReason:  ps.LastSelectReason,
			UpdatedAt:         ps.UpdatedAt,
		}
		ms := r.Store.GetMultipathStats(ctx)
		bs := r.Store.GetBulkStatus(ctx)
		if bs.AggregationMode == "" {
			bs.AggregationMode = r.Store.AggregationMode(ctx)
			bs.BulkEnabled = r.Store.BulkEnabled(ctx)
			bs.BulkMinLegs = r.Store.BulkMinHealthyLegs(ctx)
			bs.BulkMaxLegs = r.Store.EffectiveBulkMaxLegs(ctx)
		}
		out.Multipath = &api.MultipathProgress{
			Enabled:            ms.Enabled,
			Preset:             ms.Preset,
			WLEmergencyOnly:    ms.WLEmergencyOnly,
			ActiveChannels:     ms.ActiveChannels,
			HealthyChannels:    ms.HealthyChannels,
			LastReason:         ms.LastReason,
			UpdatedAt:          ms.UpdatedAt,
			AggregationMode:    bs.AggregationMode,
			BulkEnabled:        bs.BulkEnabled,
			BulkActive:         bs.BulkActive,
			BulkMemberCount:    bs.BulkMemberCount,
			BulkMinLegs:        bs.BulkMinLegs,
			BulkMaxLegs:        bs.BulkMaxLegs,
			BulkFallbackReason: bs.BulkFallbackReason,
		}
	}
	return out
}

func (r *Runtime) publishStatusEvent(typ string) {
	if r.Events == nil {
		return
	}
	st := r.statusSnapshot(context.Background())
	r.Events.Publish(api.Event{Type: typ, Status: &st})
}
