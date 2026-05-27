package mihomo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ClientOptions configures mihomo subprocess + REST API.
type ClientOptions struct {
	BinPath    string // default: MUHOMOR_MIHOMO_BIN or "mihomo"
	ConfigPath string
	ConfigDir  string
	Controller string // host:port
	Secret     string
	// APIReadyTimeout caps wait for REST /version after start; 0 = 60s (daemon), pretest uses ~12s.
	APIReadyTimeout time.Duration
}

// Client manages a mihomo child process.
type Client struct {
	opts        ClientOptions
	cmd         *exec.Cmd
	mu          sync.Mutex
	client      *http.Client
	trafficHTTP *http.Client
}

type TrafficSnapshot struct {
	Up        int64 `json:"up"`
	Down      int64 `json:"down"`
	UpTotal   int64 `json:"upTotal"`
	DownTotal int64 `json:"downTotal"`
}

func NewClient(opts ClientOptions) *Client {
	if opts.BinPath == "" {
		opts.BinPath = ResolveBin()
	}
	return &Client{
		opts:   opts,
		client: &http.Client{Timeout: 15 * time.Second},
		trafficHTTP: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd != nil && c.cmd.Process != nil {
		if c.cmd.ProcessState == nil {
			return nil
		}
		c.cmd = nil
	}
	cfgDir := c.opts.ConfigDir
	if cfgDir == "" {
		cfgDir = filepath.Dir(c.opts.ConfigPath)
	}
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return err
	}
	args := []string{"-f", c.opts.ConfigPath, "-d", cfgDir}
	// Do not use CommandContext: ctl/API callers cancel ctx after Connect returns and would kill mihomo.
	cmd := exec.Command(c.opts.BinPath, args...)
	applyCmdAttrs(cmd)
	logPath := filepath.Join(cfgDir, "mihomo-subprocess.log")
	if lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
		cmd.Stdout = lf
		cmd.Stderr = lf
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mihomo start: %w", err)
	}
	c.cmd = cmd
	go func() {
		_ = cmd.Wait()
	}()
	wait := c.opts.APIReadyTimeout
	if wait <= 0 {
		wait = 60 * time.Second
	}
	return c.waitAPI(ctx, wait)
}

func (c *Client) waitAPI(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := c.Version(ctx); err == nil {
			if c.cmd == nil || c.cmd.Process == nil || c.cmd.ProcessState != nil {
				return fmt.Errorf("mihomo exited before API ready")
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
	return fmt.Errorf("mihomo API not ready at %s", c.opts.Controller)
}

func (c *Client) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd == nil || c.cmd.Process == nil {
		return
	}
	_ = c.cmd.Process.Kill()
	_, _ = c.cmd.Process.Wait()
	c.cmd = nil
}

func (c *Client) Reload(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL()+"/configs?force=true", nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reload: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

// TrafficSnapshotNow returns the first streamed /traffic frame.
func (c *Client) TrafficSnapshotNow(ctx context.Context) (TrafficSnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.baseURL()+"/traffic", nil)
	if err != nil {
		return TrafficSnapshot{}, err
	}
	c.authorize(req)
	hc := c.trafficHTTP
	if hc == nil {
		hc = c.client
	}
	resp, err := hc.Do(req)
	if err != nil {
		return TrafficSnapshot{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return TrafficSnapshot{}, fmt.Errorf("traffic: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}
	// mihomo streams JSON frames; read one value and close (do not ReadAll the body).
	var snap TrafficSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		return TrafficSnapshot{}, err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return snap, nil
}

// TrafficNow returns current upload/download rates from GET /traffic.
func (c *Client) TrafficNow(ctx context.Context) (up, down int64, err error) {
	snap, err := c.TrafficSnapshotNow(ctx)
	if err != nil {
		return 0, 0, err
	}
	return snap.Up, snap.Down, nil
}

func (c *Client) Version(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+"/version", nil)
	if err != nil {
		return "", err
	}
	c.authorize(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("version: %s", resp.Status)
	}
	var meta struct {
		Version string `json:"version"`
		Meta    bool   `json:"meta"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", err
	}
	return meta.Version, nil
}

// TestProxyDelay implements selector.DelayTester.
func (c *Client) TestProxyDelay(ctx context.Context, proxyName string) (int, error) {
	return c.ProxyDelay(ctx, proxyName, "", 5000)
}

func (c *Client) baseURL() string {
	return "http://" + c.opts.Controller
}

func (c *Client) authorize(req *http.Request) {
	if c.opts.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.opts.Secret)
	}
}
