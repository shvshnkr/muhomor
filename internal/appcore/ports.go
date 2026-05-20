package appcore

import (
	"context"

	"github.com/muhomor/muhomor/internal/apiclient"
)

// ServiceControl — remote daemon (HTTP). Android: same interface, other transport.
type ServiceControl interface {
	Reachable(ctx context.Context) error
	Status(ctx context.Context) (apiclient.ServiceStatus, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Reload(ctx context.Context) error
	Adapt(ctx context.Context) error
	Ping(ctx context.Context) (apiclient.PingResponse, error)
	Chain(ctx context.Context, ids []int64) (apiclient.JSONResponse, error)
	ExportLog(ctx context.Context) (apiclient.JSONResponse, error)
	UpdateCheck(ctx context.Context) (apiclient.JSONResponse, error)
	UpdateInstall(ctx context.Context) (apiclient.JSONResponse, error)
	ConnectProfile(ctx context.Context, id int64) (apiclient.ServiceStatus, error)
}

// ConfigRepository — settings and profiles (remote HTTP in production UI).
type ConfigRepository interface {
	LoadSettings(ctx context.Context) (apiclient.Settings, error)
	SaveSettings(ctx context.Context, set apiclient.Settings) (apiclient.Settings, error)
	ListProfiles(ctx context.Context) ([]apiclient.Profile, error)
	ImportURI(ctx context.Context, uri string) ([]apiclient.ImportResult, error)
	RouteQuickProfile(ctx context.Context) (int, error)
	SetRouteQuickProfile(ctx context.Context, v int) error
	Close() error
}

// EventStream subscribes to daemon SSE.
type EventStream interface {
	Subscribe(ctx context.Context) (<-chan apiclient.Event, error)
}

// OutputSink — UI/platform writes user-visible text (terminal, logcat, Compose).
type OutputSink interface {
	Line(text string)
	Block(text string)
}
