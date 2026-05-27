package blexit

import (
	"context"

	"github.com/muhomor/muhomor/internal/store"
)

// RunSuiteBatches runs BL suite on candidates in TestCap batches; returns merged pass map and runner name.
func RunSuiteBatches(ctx context.Context, runners []SuiteRunner, candidates []store.Profile, st *store.Store, urls []string, minPass int) (perProfile map[int64]bool, runner string, ran bool) {
	perProfile = map[int64]bool{}
	runner = "failed"
	for i := 0; i < len(candidates); i += TestCap {
		end := i + TestCap
		if end > len(candidates) {
			end = len(candidates)
		}
		batch := candidates[i:end]
		batchPass, name, ok := runOneBatch(ctx, runners, batch, st, urls, minPass)
		if !ok {
			continue
		}
		ran = true
		runner = name
		for id, pass := range batchPass {
			if pass {
				perProfile[id] = true
			}
		}
	}
	return perProfile, runner, ran
}

func runOneBatch(ctx context.Context, runners []SuiteRunner, batch []store.Profile, st *store.Store, urls []string, minPass int) (map[int64]bool, string, bool) {
	for _, r := range runners {
		if r == nil {
			continue
		}
		m, err := r.RunSuite(ctx, batch, st, urls, minPass)
		if err != nil || len(m) == 0 {
			continue
		}
		return m, r.Name(), true
	}
	return nil, "", false
}
