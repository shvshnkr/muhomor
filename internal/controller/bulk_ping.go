package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/muhomor/muhomor/internal/api"
	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/mihomo"
	"github.com/muhomor/muhomor/internal/store"
)

// PingBulkAll probes every PROXY_BULK leg in parallel via mihomo.
func (r *Runtime) PingBulkAll(ctx context.Context) (api.BulkPingAllResponse, error) {
	st := r.Status()
	out := api.BulkPingAllResponse{
		Timestamp: time.Now().UnixMilli(),
		Connected: st.ConnectedBool(),
	}
	bs := r.Store.GetBulkStatus(ctx)
	tags := bs.BulkMemberTags
	if len(tags) == 0 {
		out.Error = "пул load-balance не активен или нет ног"
		return out, nil
	}
	r.mu.Lock()
	client := r.mihomo
	r.mu.Unlock()
	if client == nil || !st.ConnectedBool() {
		out.Error = "нет подключения к mihomo"
		return out, nil
	}
	names := r.profileNamesByID(ctx)
	results := allMemberDelays(ctx, client, tags, "", 10000)
	members, ok := memberDelaysToAPI(results, names)
	out.Members = members
	out.Total = len(members)
	out.OK = ok
	r.saveBulkMemberPings(ctx, bs, members)
	r.publishStatusEvent("bulk-ping")
	return out, nil
}

func (r *Runtime) saveBulkMemberPings(ctx context.Context, bs store.BulkStatus, members []api.BulkMemberStatus) {
	pings := make([]store.BulkMemberPing, len(members))
	for i, m := range members {
		pings[i] = store.BulkMemberPing{
			Tag: m.Tag, ProfileID: m.ProfileID, Name: m.Name,
			DelayMs: m.DelayMs, Error: m.Error,
		}
	}
	bs.MemberPings = pings
	_ = r.Store.SetBulkStatus(ctx, bs)
}

func (r *Runtime) profileNamesByID(ctx context.Context) map[int64]string {
	list, err := r.Store.ListAllProfiles(ctx)
	if err != nil {
		return nil
	}
	names := make(map[int64]string, len(list))
	for _, p := range list {
		names[p.ID] = p.Name
	}
	return names
}

func (r *Runtime) bulkMembersForStatus(ctx context.Context) []api.BulkMemberStatus {
	bs := r.Store.GetBulkStatus(ctx)
	if len(bs.BulkMemberTags) == 0 {
		return nil
	}
	pingByTag := make(map[string]store.BulkMemberPing, len(bs.MemberPings))
	for _, p := range bs.MemberPings {
		pingByTag[p.Tag] = p
	}
	names := r.profileNamesByID(ctx)
	out := make([]api.BulkMemberStatus, 0, len(bs.BulkMemberTags))
	for _, tag := range bs.BulkMemberTags {
		m := api.BulkMemberStatus{Tag: tag}
		if id, ok := configgen.ProfileIDFromBulkTag(tag); ok {
			m.ProfileID = id
			m.Name = names[id]
		}
		if p, ok := pingByTag[tag]; ok {
			m.DelayMs = p.DelayMs
			m.Error = p.Error
			if m.Name == "" {
				m.Name = p.Name
			}
		}
		out = append(out, m)
	}
	return out
}

// pingUsesBulkPool reports whether active proxy is load-balance group.
func (r *Runtime) pingUsesBulkPool(ctx context.Context) bool {
	bs := r.Store.GetBulkStatus(ctx)
	return bs.BulkActive && len(bs.BulkMemberTags) > 0
}

func (r *Runtime) pingWithBulkFallback(ctx context.Context, client *mihomo.Client, proxy string) (int, error) {
	d, err := client.TestProxyDelay(ctx, proxy)
	if err == nil && d > 0 {
		return d, nil
	}
	bs := r.Store.GetBulkStatus(ctx)
	if !bs.BulkActive || len(bs.BulkMemberTags) == 0 {
		return d, err
	}
	return bestMemberDelay(ctx, client, bs.BulkMemberTags, "", 10000)
}

func bulkPingSummary(members []api.BulkMemberStatus) string {
	if len(members) == 0 {
		return ""
	}
	ok := 0
	for _, m := range members {
		if m.DelayMs > 0 && m.Error == "" {
			ok++
		}
	}
	return fmt.Sprintf("%d/%d", ok, len(members))
}
