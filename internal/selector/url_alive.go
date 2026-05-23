package selector

import (
	"context"

	"github.com/muhomor/muhomor/internal/store"
)

func persistURLAlivePool(ctx context.Context, st *store.Store, urlDelays map[int64]int) {
	if st == nil || len(urlDelays) == 0 {
		return
	}
	ids := make([]int64, 0, len(urlDelays))
	for id, ms := range urlDelays {
		if ms > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	_ = st.SetBulkURLAlivePool(ctx, ids)
}
