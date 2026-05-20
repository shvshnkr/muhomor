package appcore

import (
	"context"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/store"
)

// ServiceControl — remote daemon (HTTP). Android: same interface, other transport.
type ServiceControl interface {
	Reachable(ctx context.Context) error
	Status(ctx context.Context) (apiclient.ServiceStatus, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Reload(ctx context.Context) error
	Adapt(ctx context.Context) error
	Chain(ctx context.Context, ids []int64) (apiclient.JSONResponse, error)
	ExportLog(ctx context.Context) (apiclient.JSONResponse, error)
	UpdateCheck(ctx context.Context) (apiclient.JSONResponse, error)
	UpdateInstall(ctx context.Context) (apiclient.JSONResponse, error)
}

// ConfigRepository — local SQLite (desktop) or ContentProvider (Android later).
type ConfigRepository interface {
	LoadSettings(ctx context.Context) (store.Settings, error)
	SaveSettings(ctx context.Context, set store.Settings) error
	ListProfiles(ctx context.Context) ([]store.Profile, error)
	ImportURI(ctx context.Context, uri string) (id int64, name string, typ string, err error)
	RouteQuickProfile(ctx context.Context) (int, error)
	SetRouteQuickProfile(ctx context.Context, v int) error
	Close() error
}

// OutputSink — UI/platform writes user-visible text (terminal, logcat, Compose).
type OutputSink interface {
	Line(text string)
	Block(text string)
}
