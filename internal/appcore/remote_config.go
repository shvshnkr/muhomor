package appcore

import (
	"context"

	"github.com/muhomor/muhomor/internal/apiclient"
)

// RemoteConfig implements ConfigRepository over daemon HTTP (Phase 4.0).
type RemoteConfig struct {
	API *apiclient.Client
}

func (r *RemoteConfig) LoadSettings(ctx context.Context) (apiclient.Settings, error) {
	return r.API.GetSettings(ctx)
}

func (r *RemoteConfig) SaveSettings(ctx context.Context, set apiclient.Settings) (apiclient.Settings, error) {
	return r.API.PutSettings(ctx, set)
}

func (r *RemoteConfig) ListProfiles(ctx context.Context) ([]apiclient.Profile, error) {
	return r.API.ListProfiles(ctx)
}

func (r *RemoteConfig) ImportURI(ctx context.Context, uri string) ([]apiclient.ImportResult, error) {
	return r.API.ImportProfiles(ctx, apiclient.ImportRequest{URI: uri})
}

func (r *RemoteConfig) ImportLines(ctx context.Context, groupID int64, lines []string) ([]apiclient.ImportResult, error) {
	req := apiclient.ImportRequest{Lines: lines}
	if groupID > 0 {
		return r.API.ImportToGroup(ctx, groupID, req)
	}
	return r.API.ImportProfiles(ctx, req)
}

func (r *RemoteConfig) RouteQuickProfile(ctx context.Context) (int, error) {
	s, err := r.API.GetSettings(ctx)
	if err != nil {
		return 0, err
	}
	return s.RouteQuickProfile, nil
}

func (r *RemoteConfig) SetRouteQuickProfile(ctx context.Context, v int) error {
	s, err := r.API.GetSettings(ctx)
	if err != nil {
		return err
	}
	s.RouteQuickProfile = v
	_, err = r.API.PutSettings(ctx, s)
	return err
}

func (r *RemoteConfig) Close() error { return nil }
