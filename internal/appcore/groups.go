package appcore

import (
	"context"

	"github.com/muhomor/muhomor/internal/apiclient"
)

// GroupsRepository manages subscription/manual groups via daemon HTTP.
type GroupsRepository interface {
	ListGroups(ctx context.Context) ([]apiclient.Group, error)
	CreateGroup(ctx context.Context, req apiclient.GroupRequest) (apiclient.Group, error)
	UpdateGroup(ctx context.Context, id int64, req apiclient.GroupRequest) error
	DeleteGroup(ctx context.Context, id int64) error
	ImportToGroup(ctx context.Context, id int64, req apiclient.ImportRequest) ([]apiclient.ImportResult, error)
	TestGroupDelays(ctx context.Context, id int64) (map[string]any, error)
	DeleteProfile(ctx context.Context, id int64) error
	TestProfileDelay(ctx context.Context, id int64) (apiclient.DelayTestResult, error)
	RefreshGroup(ctx context.Context, id int64) (map[string]any, error)
	AddGroupServer(ctx context.Context, id int64, uri string) ([]apiclient.ImportResult, error)
}

// RemoteGroups implements GroupsRepository.
type RemoteGroups struct {
	API *apiclient.Client
}

func (r *RemoteGroups) ListGroups(ctx context.Context) ([]apiclient.Group, error) {
	return r.API.ListGroups(ctx)
}

func (r *RemoteGroups) CreateGroup(ctx context.Context, req apiclient.GroupRequest) (apiclient.Group, error) {
	return r.API.CreateGroup(ctx, req)
}

func (r *RemoteGroups) UpdateGroup(ctx context.Context, id int64, req apiclient.GroupRequest) error {
	return r.API.UpdateGroup(ctx, id, req)
}

func (r *RemoteGroups) DeleteGroup(ctx context.Context, id int64) error {
	return r.API.DeleteGroup(ctx, id)
}

func (r *RemoteGroups) ImportToGroup(ctx context.Context, id int64, req apiclient.ImportRequest) ([]apiclient.ImportResult, error) {
	return r.API.ImportToGroup(ctx, id, req)
}

func (r *RemoteGroups) TestGroupDelays(ctx context.Context, id int64) (map[string]any, error) {
	return r.API.TestGroupDelays(ctx, id)
}

func (r *RemoteGroups) DeleteProfile(ctx context.Context, id int64) error {
	return r.API.DeleteProfile(ctx, id)
}

func (r *RemoteGroups) TestProfileDelay(ctx context.Context, id int64) (apiclient.DelayTestResult, error) {
	return r.API.TestProfileDelay(ctx, id)
}

func (r *RemoteGroups) RefreshGroup(ctx context.Context, id int64) (map[string]any, error) {
	return r.API.RefreshGroup(ctx, id)
}

func (r *RemoteGroups) AddGroupServer(ctx context.Context, id int64, uri string) ([]apiclient.ImportResult, error) {
	return r.API.AddGroupServer(ctx, id, uri)
}
