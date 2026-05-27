package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client talks to muhomor daemon HTTP API (Unix or TCP).
type Client struct {
	Dial DialConfig

	httpOnce    sync.Once
	transport   *http.Transport
	apiClient   *http.Client
	sseClient   *http.Client
}

// Reachable returns nil if daemon accepts HTTP.
func (c *Client) Reachable(ctx context.Context) error {
	_, err := c.Status(ctx)
	return err
}

func (c *Client) Status(ctx context.Context) (ServiceStatus, error) {
	var st ServiceStatus
	err := c.doJSON(ctx, http.MethodGet, "/v1/service/status", nil, &st)
	return st, err
}

func (c *Client) Start(ctx context.Context) error {
	return c.doOKTimeout(ctx, http.MethodPost, "/v1/service/start", nil, 10*time.Minute)
}

func (c *Client) Stop(ctx context.Context) error {
	return c.doOK(ctx, http.MethodPost, "/v1/service/stop", nil)
}

func (c *Client) Reload(ctx context.Context) error {
	return c.doOK(ctx, http.MethodPost, "/v1/service/reload", nil)
}

func (c *Client) Adapt(ctx context.Context) error {
	return c.doOK(ctx, http.MethodPost, "/v1/simple/adapt", nil)
}

func (c *Client) Chain(ctx context.Context, ids []int64) (JSONResponse, error) {
	body, _ := json.Marshal(map[string]any{"ids": ids})
	var out JSONResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/service/chain", body, &out)
	return out, err
}

func (c *Client) ExportLog(ctx context.Context) (JSONResponse, error) {
	var out JSONResponse
	err := c.doJSON(ctx, http.MethodGet, "/v1/logs/export", nil, &out)
	return out, err
}

func (c *Client) UpdateCheck(ctx context.Context) (JSONResponse, error) {
	var out JSONResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/update/check", nil, &out)
	return out, err
}

func (c *Client) UpdateInstall(ctx context.Context) (JSONResponse, error) {
	var out JSONResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/update/install", nil, &out)
	return out, err
}

func (c *Client) doOK(ctx context.Context, method, path string, body []byte) error {
	return c.doJSON(ctx, method, path, body, nil)
}

func (c *Client) doOKTimeout(ctx context.Context, method, path string, body []byte, timeout time.Duration) error {
	return c.doJSONTimeout(ctx, method, path, body, nil, timeout)
}

func (c *Client) doJSON(ctx context.Context, method, path string, body []byte, out any) error {
	return c.doJSONTimeout(ctx, method, path, body, out, 0)
}

func (c *Client) doJSONTimeout(ctx context.Context, method, path string, body []byte, out any, timeout time.Duration) error {
	var r io.Reader
	if len(body) > 0 {
		r = bytes.NewReader(body)
	}
	reqCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(reqCtx, method, "http://localhost"+path, r)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.apiHTTP().Do(req)
	if err != nil {
		return wrapDaemonErr(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	if out == nil {
		return nil
	}
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}
