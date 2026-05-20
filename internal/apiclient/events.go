package apiclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// StreamEvents subscribes to GET /v1/events (SSE).
func (c *Client) StreamEvents(ctx context.Context) (<-chan Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/v1/events", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	client := &http.Client{
		Transport: &http.Transport{DialContext: c.Dial.DialContext},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("daemon not running: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		resp.Body.Close()
		return nil, fmt.Errorf("events: %s", resp.Status)
	}
	out := make(chan Event, 16)
	go func() {
		defer resp.Body.Close()
		defer close(out)
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var ev Event
			if json.Unmarshal([]byte(line[6:]), &ev) == nil {
				select {
				case out <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

// Ping calls POST /v1/service/ping.
func (c *Client) Ping(ctx context.Context) (PingResponse, error) {
	var out PingResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/service/ping", nil, &out)
	return out, err
}

// GetSettings GET /v1/settings.
func (c *Client) GetSettings(ctx context.Context) (Settings, error) {
	var out Settings
	err := c.doJSON(ctx, http.MethodGet, "/v1/settings", nil, &out)
	return out, err
}

// PutSettings PUT /v1/settings.
func (c *Client) PutSettings(ctx context.Context, s Settings) (Settings, error) {
	body, _ := json.Marshal(s)
	var out Settings
	err := c.doJSON(ctx, http.MethodPut, "/v1/settings", body, &out)
	return out, err
}

// ListProfiles GET /v1/profiles.
func (c *Client) ListProfiles(ctx context.Context) ([]Profile, error) {
	var wrap struct {
		Profiles []Profile `json:"profiles"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/v1/profiles", nil, &wrap); err != nil {
		return nil, err
	}
	return wrap.Profiles, nil
}

// ImportProfiles POST /v1/profiles/import.
func (c *Client) ImportProfiles(ctx context.Context, req ImportRequest) ([]ImportResult, error) {
	body, _ := json.Marshal(req)
	var wrap struct {
		Results []ImportResult `json:"results"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/v1/profiles/import", body, &wrap); err != nil {
		return nil, err
	}
	return wrap.Results, nil
}

// SetProfileEnabled PUT /v1/profiles/{id}/enabled.
func (c *Client) SetProfileEnabled(ctx context.Context, id int64, enabled bool) error {
	body, _ := json.Marshal(ProfileEnabledReq{Enabled: enabled})
	return c.doOK(ctx, http.MethodPut, fmt.Sprintf("/v1/profiles/%d/enabled", id), body)
}

// ConnectProfile POST /v1/profiles/{id}/connect.
func (c *Client) ConnectProfile(ctx context.Context, id int64) (ServiceStatus, error) {
	var st ServiceStatus
	err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/v1/profiles/%d/connect", id), nil, &st)
	return st, err
}
