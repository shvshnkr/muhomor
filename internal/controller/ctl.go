package controller

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/muhomor/muhomor/internal/paths"
)

// RunCtl executes Desktop-compatible --ctl against a running daemon.
func RunCtl(layout paths.Layout, command string) error {
	cmd := strings.TrimSpace(strings.ToLower(command))
	switch cmd {
	case "start", "stop", "reload":
		method := http.MethodPost
		path := "/v1/service/" + cmd
		return ctlHTTP(layout.SocketPath(), method, path)
	case "status":
		if err := ctlHTTP(layout.SocketPath(), http.MethodGet, "/v1/service/status"); err != nil {
			return err
		}
		b, err := os.ReadFile(layout.ControlStatusFile())
		if err != nil {
			return err
		}
		fmt.Print(string(b))
		return nil
	case "ping":
		ping := layout.CacheDir + string(os.PathSeparator) + "desktop-control-ping.txt"
		return os.WriteFile(ping, []byte(fmt.Sprintf("timestamp=%d\n", time.Now().UnixMilli())), 0o644)
	case "export-log":
		return ctlHTTP(layout.SocketPath(), http.MethodGet, "/v1/logs/export")
	case "update-check":
		return ctlHTTP(layout.SocketPath(), http.MethodPost, "/v1/update/check")
	case "update-install":
		return ctlHTTP(layout.SocketPath(), http.MethodPost, "/v1/update/install")
	default:
		return fmt.Errorf("unknown --ctl command: %s", command)
	}
}

func ctlHTTP(socketPath, method, path string) error {
	tr := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			if _, err := os.Stat(socketPath); err == nil {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			}
			return (&net.Dialer{}).DialContext(ctx, "tcp", "127.0.0.1:8751")
		},
	}
	client := &http.Client{Timeout: 60 * time.Second, Transport: tr}
	req, err := http.NewRequest(method, "http://localhost"+path, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("daemon not running: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}
