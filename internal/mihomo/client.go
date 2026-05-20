package mihomo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/muhomor/muhomor/internal/reachability"
)

// ClientOptions configures mihomo subprocess + REST API.
type ClientOptions struct {
	BinPath      string // default: MUHOMOR_MIHOMO_BIN or "mihomo"
	ConfigPath   string
	ConfigDir    string
	Controller   string // host:port
	Secret       string
}

// Client manages a mihomo child process.
type Client struct {
	opts   ClientOptions
	cmd    *exec.Cmd
	mu     sync.Mutex
	client *http.Client
}

func NewClient(opts ClientOptions) *Client {
	if opts.BinPath == "" {
		opts.BinPath = ResolveBin()
	}
	return &Client{
		opts: opts,
		client: &http.Client{Timeout: 15 * time.Second},
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
	return c.waitAPI(ctx, 60*time.Second)
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

func (c *Client) ProxyDelay(ctx context.Context, proxyName string, testURL string, timeoutMs int) (int, error) {
	if testURL == "" {
		testURL = reachability.ConnectionTestURL
	}
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	u := fmt.Sprintf("%s/proxies/%s/delay?timeout=%d&url=%s",
		c.baseURL(), url.PathEscape(proxyName), timeoutMs, url.QueryEscape(testURL))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	c.authorize(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var out struct {
		Delay int `json:"delay"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	return out.Delay, nil
}

func (c *Client) baseURL() string {
	return "http://" + c.opts.Controller
}

func (c *Client) authorize(req *http.Request) {
	if c.opts.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.opts.Secret)
	}
}
