package controller

import (
	"context"
	"sync"

	"github.com/muhomor/muhomor/internal/api"
	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/mihomo"
	"github.com/muhomor/muhomor/internal/reachability"
)

const bulkDelayWorkers = 16

// memberDelayResult is one parallel proxy delay probe.
type memberDelayResult struct {
	Tag     string
	DelayMs int
	Err     error
}

// postConnectDelay checks connectivity after mihomo start.
// For PROXY_BULK, mihomo often returns 503 on the group delay API; fall back to member tags.
func postConnectDelay(ctx context.Context, client *mihomo.Client, plan configgen.BulkPlan, primaryProxy, testURL string, testMs int) (effective string, delay int, err error) {
	effective = primaryProxy
	if plan.Active && plan.MatchTarget != "" {
		effective = plan.MatchTarget
	}
	delay, err = client.ProxyDelay(ctx, effective, testURL, testMs)
	if err == nil && delay > 0 {
		return effective, delay, nil
	}
	if !plan.Active || len(plan.BulkTags) == 0 {
		return effective, delay, err
	}
	best, memberErr := bestMemberDelay(ctx, client, plan.BulkTags, testURL, testMs)
	if best > 0 {
		return effective, best, nil
	}
	if memberErr != nil {
		return effective, 0, memberErr
	}
	return effective, delay, err
}

func allMemberDelays(ctx context.Context, client *mihomo.Client, tags []string, testURL string, testMs int) []memberDelayResult {
	if testURL == "" {
		testURL = reachability.ConnectionTestURL
	}
	if testMs <= 0 {
		testMs = 10000
	}
	out := make([]memberDelayResult, len(tags))
	sem := make(chan struct{}, bulkDelayWorkers)
	var wg sync.WaitGroup
	for i, tag := range tags {
		i, tag := i, tag
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			d, e := client.ProxyDelay(ctx, tag, testURL, testMs)
			out[i] = memberDelayResult{Tag: tag, DelayMs: d, Err: e}
		}()
	}
	wg.Wait()
	return out
}

func bestMemberDelay(ctx context.Context, client *mihomo.Client, tags []string, testURL string, testMs int) (int, error) {
	best := 0
	var lastErr error
	for _, r := range allMemberDelays(ctx, client, tags, testURL, testMs) {
		if r.Err == nil && r.DelayMs > 0 {
			if best == 0 || r.DelayMs < best {
				best = r.DelayMs
			}
			continue
		}
		if r.Err != nil {
			lastErr = r.Err
		}
	}
	if best > 0 {
		return best, nil
	}
	return 0, lastErr
}

func memberDelaysToAPI(results []memberDelayResult, names map[int64]string) ([]api.BulkMemberStatus, int) {
	out := make([]api.BulkMemberStatus, len(results))
	ok := 0
	for i, r := range results {
		m := api.BulkMemberStatus{Tag: r.Tag}
		if id, okID := configgen.ProfileIDFromBulkTag(r.Tag); okID {
			m.ProfileID = id
			m.Name = names[id]
		}
		if r.Err == nil && r.DelayMs > 0 {
			m.DelayMs = r.DelayMs
			ok++
		} else if r.Err != nil {
			m.Error = r.Err.Error()
		} else {
			m.Error = "нет ответа"
		}
		out[i] = m
	}
	return out, ok
}
