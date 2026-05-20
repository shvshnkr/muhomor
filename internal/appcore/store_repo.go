package appcore

import (
	"context"
	"fmt"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

// LocalStoreRepo implements ConfigRepository on SQLite.
type LocalStoreRepo struct {
	Store *store.Store
}

func (r *LocalStoreRepo) LoadSettings(ctx context.Context) (store.Settings, error) {
	return r.Store.LoadSettings(ctx)
}

func (r *LocalStoreRepo) SaveSettings(ctx context.Context, set store.Settings) error {
	return r.Store.SaveSettings(ctx, set)
}

func (r *LocalStoreRepo) ListProfiles(ctx context.Context) ([]store.Profile, error) {
	return r.Store.ListAllProfiles(ctx)
}

func (r *LocalStoreRepo) RouteQuickProfile(ctx context.Context) (int, error) {
	return r.Store.RouteQuickProfile(ctx)
}

func (r *LocalStoreRepo) SetRouteQuickProfile(ctx context.Context, v int) error {
	return r.Store.SetRouteQuickProfile(ctx, v)
}

func (r *LocalStoreRepo) Close() error {
	return r.Store.Close()
}

func (r *LocalStoreRepo) ImportURI(ctx context.Context, uri string) (int64, string, string, error) {
	if reason := subscription.UnsupportedReason(uri); reason != "" {
		return 0, "", "", fmt.Errorf("unsupported: %s", reason)
	}
	typ := subscription.Scheme(uri)
	name := "imported"
	if typ == "vless" {
		if p, err := configgen.ParseVLESSURI(uri); err == nil && p.Name != "" {
			name = p.Name
		}
	}
	id, err := r.Store.UpsertProfile(ctx, name, typ, uri)
	return id, name, typ, err
}
