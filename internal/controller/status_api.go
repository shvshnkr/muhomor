package controller

import (
	"context"
	"strconv"
	"strings"

	"time"

	"github.com/muhomor/muhomor/internal/api"
	"github.com/muhomor/muhomor/internal/standby"
	"github.com/muhomor/muhomor/internal/store"
)

func isStartupActivity(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	return strings.HasPrefix(text, "Запуск mihomo") ||
		strings.HasPrefix(text, "Подключение") ||
		strings.HasPrefix(text, "Переподключение") ||
		strings.HasPrefix(text, "Обновление конфигурации") ||
		strings.HasPrefix(text, "Проверка соединения") ||
		strings.HasPrefix(text, "Сервер нестабилен")
}

func (r *Runtime) statusSnapshot(ctx context.Context) api.ServiceStatus {
	st := r.Status()
	activity, _ := r.Store.GetKV(ctx, store.KeySimpleModeActivity)
	lastPing, _ := r.Store.GetKV(ctx, store.KeyLastServicePingMs)
	lastPingErr, _ := r.Store.GetKV(ctx, store.KeyLastServicePingError)
	out := api.ServiceStatus{
		State:              api.ServiceState(st.State),
		Connected:          st.ConnectedBool(),
		ProfileID:          st.ProfileID,
		ProfileName:        st.ProfileName,
		ProxyName:          st.ProxyName,
		SubscriptionSource: st.SubscriptionSource,
		ActivityText:       activity,
		LastPingError:      lastPingErr,
		BulkMembers:        r.bulkMembersForStatus(ctx),
	}
	vs := VerificationSnapshot{}
	if r.verifier != nil {
		vs = r.verifier.Snapshot()
	}
	out.ConnectedVerified = vs.ConnectedVerified
	out.ConnectedDegraded = vs.ConnectedDegraded
	out.LiveServersConfirmed = vs.LiveServersConfirmed
	out.VerificationPhase = api.VerificationPhase(vs.Phase)
	out.VerificationReason = vs.Reason
	out.LiveServerEvidence = &api.LiveServerEvidence{
		TCPOK:            vs.Evidence.TCPOK,
		URLOK:            vs.Evidence.URLOK,
		PoolSize:         vs.Evidence.PoolSize,
		Degraded:         vs.Evidence.Degraded,
		LastSuccessAtUTC: vs.Evidence.LastSuccessAtUTC,
	}
	if n, err := strconv.Atoi(lastPing); err == nil && n > 0 {
		out.LastPingMs = n
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
		out.Standby = r.standbyProgress(ctx)
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
	return r.reconcileServiceStatus(ctx, out)
}

func (r *Runtime) standbyProgress(ctx context.Context) *api.StandbyProgress {
	if r.Store == nil {
		return nil
	}
	sp := &api.StandbyProgress{
		PickerRunning: r.picker != nil && r.picker.Available(),
	}
	for _, e := range standby.LoadHot(ctx, r.Store) {
		sp.Hot = append(sp.Hot, apiStandbyEntry(e))
	}
	for _, e := range standby.LoadWarm(ctx, r.Store) {
		sp.Warm = append(sp.Warm, apiStandbyEntry(e))
	}
	if t, ok := standby.LastRefreshAt(ctx, r.Store); ok {
		sp.LastRefreshAt = t.UTC().Format(time.RFC3339)
	}
	return sp
}

func apiStandbyEntry(e standby.Entry) api.StandbyEntry {
	out := api.StandbyEntry{
		ProfileID: e.ProfileID,
		ProxyName: e.ProxyName,
		DelayMs:   e.DelayMs,
	}
	if !e.VerifiedAt.IsZero() {
		out.VerifiedAt = e.VerifiedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func (r *Runtime) reconcileServiceStatus(ctx context.Context, out api.ServiceStatus) api.ServiceStatus {
	r.mu.Lock()
	client := r.mihomo
	proxy := r.proxy
	mem := r.status
	r.mu.Unlock()

	if client != nil && proxy != "" {
		if out.ConnectedVerified {
			out.State = api.StateConnected
			out.Connected = true
		} else {
			out.State = api.StateConnecting
			out.Connected = false
		}
		out.ProxyName = proxy
		out.TrafficUp, out.TrafficDown = r.cachedTraffic()
		if out.ProfileID == 0 {
			if id, err := r.Store.CurrentProfileID(ctx); err == nil && id > 0 {
				out.ProfileID = id
				if p, err := r.Store.ProfileByID(ctx, id); err == nil {
					out.ProfileName = p.Name
				}
			}
		}
		if out.ProfileID == 0 && mem.ProfileID > 0 {
			out.ProfileID = mem.ProfileID
			out.ProfileName = mem.ProfileName
		}
		if isStartupActivity(out.ActivityText) {
			_ = r.Store.SetKV(ctx, store.KeySimpleModeActivity, "")
			out.ActivityText = ""
		}
		if out.ConnectedVerified && (mem.State != StateConnected || mem.ProxyName != proxy) {
			r.patchStatus(Status{
				State:       StateConnected,
				Connected:   true,
				ProfileID:   out.ProfileID,
				ProfileName: out.ProfileName,
				ProxyName:   proxy,
			})
		}
		return out
	}
	if out.VerificationPhase == api.VerificationNoLiveServers {
		out.State = api.StateIdle
		out.Connected = false
	}

	if !r.connectInFlight() && out.State == api.StateConnecting && !out.Connected && isStartupActivity(out.ActivityText) {
		_ = r.Store.SetKV(ctx, store.KeySimpleModeActivity, "")
		out.ActivityText = ""
		out.State = api.StateIdle
		out.LastPingMs = 0
		out.LastPingError = ""
		_ = r.Store.SetKV(ctx, store.KeyLastServicePingMs, "")
		_ = r.Store.SetKV(ctx, store.KeyLastServicePingError, "")
		if mem.State == StateConnecting {
			r.patchStatus(Status{State: StateIdle})
		}
	}
	if !out.Connected {
		out.LastPingMs = 0
		out.LastPingError = ""
		_ = r.Store.SetKV(ctx, store.KeyLastServicePingError, "")
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
