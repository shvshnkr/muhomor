package controller

import (
	"context"

	"github.com/muhomor/muhomor/internal/api"
	"github.com/muhomor/muhomor/internal/store"
)

func (r *Runtime) statusSnapshot(ctx context.Context) api.ServiceStatus {
	st := r.Status()
	activity, _ := r.Store.GetKV(ctx, store.KeySimpleModeActivity)
	return api.ServiceStatus{
		State:              api.ServiceState(st.State),
		Connected:          st.ConnectedBool(),
		ProfileID:          st.ProfileID,
		ProfileName:        st.ProfileName,
		ProxyName:          st.ProxyName,
		SubscriptionSource: st.SubscriptionSource,
		ActivityText:       activity,
	}
}

func (r *Runtime) publishStatusEvent(typ string) {
	if r.Events == nil {
		return
	}
	st := r.statusSnapshot(context.Background())
	r.Events.Publish(api.Event{Type: typ, Status: &st})
}
