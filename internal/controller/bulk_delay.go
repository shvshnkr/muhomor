package controller

import (
	"context"
	"sync"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/mihomo"
)

const bulkDelayWorkers = 16

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

func bestMemberDelay(ctx context.Context, client *mihomo.Client, tags []string, testURL string, testMs int) (int, error) {
	best := 0
	var lastErr error
	var mu sync.Mutex
	sem := make(chan struct{}, bulkDelayWorkers)
	var wg sync.WaitGroup
	for _, tag := range tags {
		tag := tag
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			d, e := client.ProxyDelay(ctx, tag, testURL, testMs)
			if e == nil && d > 0 {
				mu.Lock()
				if best == 0 || d < best {
					best = d
				}
				mu.Unlock()
				return
			}
			if e != nil {
				mu.Lock()
				lastErr = e
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if best > 0 {
		return best, nil
	}
	return 0, lastErr
}
