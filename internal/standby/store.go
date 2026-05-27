package standby

import (
	"context"
	"encoding/json"
	"time"

	"github.com/muhomor/muhomor/internal/store"
)

func loadEntries(ctx context.Context, st *store.Store, key string) []Entry {
	if st == nil {
		return nil
	}
	raw, _ := st.GetKV(ctx, key)
	if raw == "" {
		return nil
	}
	var entries []Entry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil
	}
	return entries
}

func saveEntries(ctx context.Context, st *store.Store, key string, entries []Entry) error {
	if st == nil {
		return nil
	}
	if entries == nil {
		entries = []Entry{}
	}
	b, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	return st.SetKV(ctx, key, string(b))
}

// LoadHot reads hot standby pool from KV.
func LoadHot(ctx context.Context, st *store.Store) []Entry {
	return loadEntries(ctx, st, store.KeyHotStandby)
}

// LoadWarm reads warm standby pool from KV.
func LoadWarm(ctx context.Context, st *store.Store) []Entry {
	return loadEntries(ctx, st, store.KeyWarmStandby)
}

// SaveHot persists hot standby pool.
func SaveHot(ctx context.Context, st *store.Store, entries []Entry) error {
	return saveEntries(ctx, st, store.KeyHotStandby, entries)
}

// SaveWarm persists warm standby pool.
func SaveWarm(ctx context.Context, st *store.Store, entries []Entry) error {
	return saveEntries(ctx, st, store.KeyWarmStandby, entries)
}

// TouchLastRefresh records standby refresher activity time.
func TouchLastRefresh(ctx context.Context, st *store.Store) {
	if st == nil {
		return
	}
	_ = st.SetKV(ctx, store.KeyStandbyLastRefreshAt, time.Now().UTC().Format(time.RFC3339))
}

// LastRefreshAt returns last refresher tick time if set.
func LastRefreshAt(ctx context.Context, st *store.Store) (time.Time, bool) {
	if st == nil {
		return time.Time{}, false
	}
	raw, _ := st.GetKV(ctx, store.KeyStandbyLastRefreshAt)
	if raw == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
