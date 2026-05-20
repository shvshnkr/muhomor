package appcore

import (
	"context"

	"github.com/muhomor/muhomor/internal/apiclient"
)

// DaemonClient implements ServiceControl via apiclient.
type DaemonClient struct {
	API *apiclient.Client
}

func (d *DaemonClient) Reachable(ctx context.Context) error {
	return d.API.Reachable(ctx)
}

func (d *DaemonClient) Status(ctx context.Context) (apiclient.ServiceStatus, error) {
	return d.API.Status(ctx)
}

func (d *DaemonClient) Start(ctx context.Context) error {
	return d.API.Start(ctx)
}

func (d *DaemonClient) Stop(ctx context.Context) error {
	return d.API.Stop(ctx)
}

func (d *DaemonClient) Reload(ctx context.Context) error {
	return d.API.Reload(ctx)
}

func (d *DaemonClient) Adapt(ctx context.Context) error {
	return d.API.Adapt(ctx)
}

func (d *DaemonClient) Chain(ctx context.Context, ids []int64) (apiclient.JSONResponse, error) {
	return d.API.Chain(ctx, ids)
}

func (d *DaemonClient) ExportLog(ctx context.Context) (apiclient.JSONResponse, error) {
	return d.API.ExportLog(ctx)
}

func (d *DaemonClient) UpdateCheck(ctx context.Context) (apiclient.JSONResponse, error) {
	return d.API.UpdateCheck(ctx)
}

func (d *DaemonClient) UpdateInstall(ctx context.Context) (apiclient.JSONResponse, error) {
	return d.API.UpdateInstall(ctx)
}
