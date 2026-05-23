package mihomo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/muhomor/muhomor/internal/reachability"
)

type delayResponse struct {
	Delay   int    `json:"delay"`
	Message string `json:"message"`
}

// ProxyDelay tests one proxy via GET /proxies/{name}/delay.
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
	resp, err := c.doDelayRequest(req, time.Duration(timeoutMs)*time.Millisecond+5*time.Second)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	out, err := parseDelayBody(resp)
	if err != nil {
		return 0, err
	}
	if out.Delay <= 0 {
		if out.Message != "" {
			return 0, fmt.Errorf("%s", out.Message)
		}
		return 0, fmt.Errorf("proxy delay test returned 0")
	}
	return out.Delay, nil
}

// GroupDelay tests all proxies in a url-test group via GET /group/{name}/delay.
func (c *Client) GroupDelay(ctx context.Context, groupName string, testURL string, timeoutMs int) (map[string]int, error) {
	if testURL == "" {
		testURL = reachability.ConnectionTestURL
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	u := fmt.Sprintf("%s/group/%s/delay?timeout=%d&url=%s",
		c.baseURL(), url.PathEscape(groupName), timeoutMs, url.QueryEscape(testURL))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	// HTTP client must outlive mihomo's group context (timeoutMs is whole-group deadline).
	httpWait := time.Duration(timeoutMs)*time.Millisecond + 20*time.Second
	resp, err := c.doDelayRequest(req, httpWait)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("group delay: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var raw map[string]uint16
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("group delay json: %w", err)
	}
	out := make(map[string]int, len(raw))
	for k, v := range raw {
		if v > 0 {
			out[k] = int(v)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("group delay: no live proxies (%s)", strings.TrimSpace(string(body)))
	}
	return out, nil
}

func (c *Client) doDelayRequest(req *http.Request, timeout time.Duration) (*http.Response, error) {
	client := c.client
	if timeout > 0 {
		client = &http.Client{Timeout: timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusGatewayTimeout {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("delay timeout: %s", strings.TrimSpace(string(body)))
	}
	return resp, nil
}

func parseDelayBody(resp *http.Response) (delayResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return delayResponse{}, err
	}
	if resp.StatusCode/100 != 2 {
		return delayResponse{}, fmt.Errorf("delay: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out delayResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return delayResponse{}, err
	}
	return out, nil
}
