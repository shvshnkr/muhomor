package selector

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/store"
)

const (
	pretestParallelWorkers   = 24
	pretestMicroBatchSize    = 5
	pretestMicroBatchWorkers = 12
	pretestSingleAPIWait     = 12 * time.Second

	pretestParallelWorkersWSL   = 8
	pretestMicroBatchWorkersWSL = 4
)

func pretestParallelLimit() int {
	if runtime.GOOS == "linux" && paths.IsWSL() {
		return pretestParallelWorkersWSL
	}
	return pretestParallelWorkers
}

func pretestMicroBatchLimit() int {
	if runtime.GOOS == "linux" && paths.IsWSL() {
		return pretestMicroBatchWorkersWSL
	}
	return pretestMicroBatchWorkers
}

// TestProfilesBatch URL-tests candidates (picker batch primary; parallel+micro emergency fallback).
func (e *EphemeralTester) TestProfilesBatch(ctx context.Context, profiles []store.Profile) map[int64]int {
	if len(profiles) == 0 {
		return nil
	}
	if out := e.testProfilesPickerWithCatchup(ctx, profiles); len(out) > 0 {
		return out
	}
	// Emergency fallback: micro-batch only, cap 12 (no 24× parallel on happy path).
	if len(profiles) > urlTestCapPicker {
		profiles = profiles[:urlTestCapPicker]
	}
	if e.Picker != nil && pickerReady(ctx, e.Picker) {
		return e.testProfilesMicroBatchParallel(ctx, profiles)
	}
	workers := pretestParallelLimit()
	if e.Activity != nil {
		e.Activity(ctx, fmt.Sprintf("URL тест %d серверов (%d параллельно)…", len(profiles), workers))
	}
	out := e.testProfilesParallel(ctx, profiles, workers)
	if len(out) < len(profiles) {
		var missing []store.Profile
		for _, p := range profiles {
			if out[p.ID] <= 0 {
				missing = append(missing, p)
			}
		}
		if len(missing) > 0 {
			if e.Activity != nil {
				e.Activity(ctx, fmt.Sprintf("URL догон %d серверов (micro-batch)…", len(missing)))
			}
			part := e.testProfilesMicroBatchParallel(ctx, missing)
			for id, ms := range part {
				if ms > 0 {
					out[id] = ms
				}
			}
		}
	}
	if e.Log != nil {
		fail := len(profiles) - len(out)
		e.Log.Info("ephemeral pretest", "ok", len(out), "fail", fail, "candidates", len(profiles), "event", "H4-parallel")
	}
	if e.Activity != nil {
		e.Activity(ctx, fmt.Sprintf("URL тест готов: ok=%d fail=%d", len(out), len(profiles)-len(out)))
	}
	return out
}

func (e *EphemeralTester) testProfilesParallel(ctx context.Context, profiles []store.Profile, workers int) map[int64]int {
	if workers <= 0 {
		workers = pretestParallelLimit()
	}
	if workers > len(profiles) {
		workers = len(profiles)
	}
	out := make(map[int64]int)
	var mu sync.Mutex
	var done atomic.Int32
	total := int32(len(profiles))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for _, p := range profiles {
		if ctx.Err() != nil {
			break
		}
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			ms, err := e.testProfileTimed(ctx, p)
			if err == nil && ms > 0 {
				mu.Lock()
				out[p.ID] = ms
				mu.Unlock()
			}
			n := done.Add(1)
			if e.Activity != nil && (n == total || n%6 == 0) {
				mu.Lock()
				ok := len(out)
				mu.Unlock()
				e.Activity(ctx, fmt.Sprintf("URL тест %d/%d (живых %d)…", n, total, ok))
			}
		}()
	}
	wg.Wait()
	return out
}

func (e *EphemeralTester) testProfileTimed(ctx context.Context, p store.Profile) (int, error) {
	perProxyMs := 8000
	if e.Store != nil {
		if t := e.Store.ConnectionTestTimeoutMs(ctx); t > 0 {
			perProxyMs = t
		}
	}
	budget := pretestSingleAPIWait + time.Duration(perProxyMs)*time.Millisecond + 8*time.Second
	tctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	return e.testProfileWithTimeout(tctx, p, perProxyMs)
}

func (e *EphemeralTester) testProfilesMicroBatchParallel(ctx context.Context, profiles []store.Profile) map[int64]int {
	var chunks [][]store.Profile
	for i := 0; i < len(profiles); i += pretestMicroBatchSize {
		end := i + pretestMicroBatchSize
		if end > len(profiles) {
			end = len(profiles)
		}
		chunks = append(chunks, profiles[i:end])
	}
	out := make(map[int64]int)
	var mu sync.Mutex
	sem := make(chan struct{}, pretestMicroBatchLimit())
	var wg sync.WaitGroup
	for _, chunk := range chunks {
		chunk := chunk
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			part := e.testProfilesMicroBatchOnce(ctx, chunk)
			if len(part) == 0 {
				return
			}
			mu.Lock()
			for id, ms := range part {
				out[id] = ms
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}
