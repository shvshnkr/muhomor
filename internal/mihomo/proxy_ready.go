package mihomo

import (
	"context"
	"strings"
	"time"
)

// WaitProxyReady polls until the proxy answers delay API or maxWait elapses.
// Transient 503/refused during mihomo reload are retried; other errors mean the proxy is registered.
func (c *Client) WaitProxyReady(ctx context.Context, proxyName string, maxWait time.Duration) error {
	if proxyName == "" || maxWait <= 0 {
		return nil
	}
	deadline := time.Now().Add(maxWait)
	probeMs := 2500
	for time.Now().Before(deadline) {
		_, err := c.ProxyDelay(ctx, proxyName, "", probeMs)
		if err == nil {
			return nil
		}
		if !proxyNotReadyErr(err) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}
	}
	return nil
}

func proxyNotReadyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "an error occurred in the delay test") ||
		strings.Contains(msg, "not found")
}
