package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/muhomor/muhomor/internal/api"
)

type (
	Group           = api.Group
	GroupRequest    = api.GroupRequest
	DelayTestResult = api.DelayTestResult
)

// ListGroups GET /v1/groups.
func (c *Client) ListGroups(ctx context.Context) ([]Group, error) {
	var wrap struct {
		Groups []Group `json:"groups"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/v1/groups", nil, &wrap); err != nil {
		return nil, err
	}
	return wrap.Groups, nil
}

// CreateGroup POST /v1/groups.
func (c *Client) CreateGroup(ctx context.Context, req GroupRequest) (Group, error) {
	body, _ := json.Marshal(req)
	var g Group
	if err := c.doJSON(ctx, http.MethodPost, "/v1/groups", body, &g); err != nil {
		return g, err
	}
	return g, nil
}

// UpdateGroup PUT /v1/groups/{id}.
func (c *Client) UpdateGroup(ctx context.Context, id int64, req GroupRequest) error {
	body, _ := json.Marshal(req)
	return c.doOK(ctx, http.MethodPut, fmt.Sprintf("/v1/groups/%d", id), body)
}

// DeleteGroup DELETE /v1/groups/{id}.
func (c *Client) DeleteGroup(ctx context.Context, id int64) error {
	return c.doOK(ctx, http.MethodDelete, fmt.Sprintf("/v1/groups/%d", id), nil)
}

// ImportToGroup POST /v1/groups/{id}/import.
func (c *Client) ImportToGroup(ctx context.Context, id int64, req ImportRequest) ([]ImportResult, error) {
	body, _ := json.Marshal(req)
	var wrap struct {
		Results []ImportResult `json:"results"`
	}
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/v1/groups/%d/import", id), body, &wrap); err != nil {
		return nil, err
	}
	return wrap.Results, nil
}

// TestGroupDelays POST /v1/groups/{id}/test-all (may take minutes).
func (c *Client) TestGroupDelays(ctx context.Context, id int64) (map[string]any, error) {
	var out map[string]any
	err := c.doJSONTimeout(ctx, http.MethodPost, fmt.Sprintf("/v1/groups/%d/test-all", id), nil, &out, 15*time.Minute)
	return out, err
}

// DeleteProfile DELETE /v1/profiles/{id}.
func (c *Client) DeleteProfile(ctx context.Context, id int64) error {
	return c.doOK(ctx, http.MethodDelete, fmt.Sprintf("/v1/profiles/%d", id), nil)
}

// TestProfileDelay POST /v1/profiles/{id}/delay-test.
func (c *Client) TestProfileDelay(ctx context.Context, id int64) (DelayTestResult, error) {
	var out DelayTestResult
	err := c.doJSONTimeout(ctx, http.MethodPost, fmt.Sprintf("/v1/profiles/%d/delay-test", id), nil, &out, 3*time.Minute)
	return out, err
}

// RefreshGroup POST /v1/groups/{id}/refresh (subscription groups).
func (c *Client) RefreshGroup(ctx context.Context, id int64) (map[string]any, error) {
	var out map[string]any
	err := c.doJSONTimeout(ctx, http.MethodPost, fmt.Sprintf("/v1/groups/%d/refresh", id), nil, &out, 5*time.Minute)
	return out, err
}

// AddGroupServer POST /v1/groups/{id}/servers (manual groups).
func (c *Client) AddGroupServer(ctx context.Context, id int64, uri string) ([]ImportResult, error) {
	body, _ := json.Marshal(ImportRequest{URI: uri})
	var wrap struct {
		Results []ImportResult `json:"results"`
	}
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/v1/groups/%d/servers", id), body, &wrap); err != nil {
		return nil, err
	}
	return wrap.Results, nil
}

// ShutdownDaemon POST /v1/daemon/shutdown (stops daemon process).
func (c *Client) ShutdownDaemon(ctx context.Context) error {
	return c.doOK(ctx, http.MethodPost, "/v1/daemon/shutdown", nil)
}
