package controller

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/paths"
)

// RunCtl executes Desktop-compatible --ctl against a running daemon.
func RunCtl(layout paths.Layout, command string) error {
	cmd := strings.TrimSpace(strings.ToLower(command))
	switch cmd {
	case "start", "stop", "reload":
		return ctlHTTP(layout, http.MethodPost, "/v1/service/"+cmd)
	case "status":
		if err := ctlHTTP(layout, http.MethodGet, "/v1/service/status"); err != nil {
			return err
		}
		b, err := os.ReadFile(layout.ControlStatusFile())
		if err != nil {
			return err
		}
		fmt.Print(string(b))
		return nil
	case "ping":
		return ctlHTTP(layout, http.MethodPost, "/v1/service/ping")
	case "export-log":
		return ctlHTTP(layout, http.MethodGet, "/v1/logs/export")
	case "update-check":
		return ctlHTTP(layout, http.MethodPost, "/v1/update/check")
	case "update-install":
		return ctlHTTP(layout, http.MethodPost, "/v1/update/install")
	default:
		return fmt.Errorf("unknown --ctl command: %s", command)
	}
}

func ctlHTTP(layout paths.Layout, method, path string) error {
	dial := apiclient.DialConfig{SocketPath: layout.SocketPath()}
	tr := &http.Transport{DialContext: dial.DialContext}
	client := &http.Client{Timeout: 120 * time.Second, Transport: tr}
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
	if method == http.MethodGet || method == http.MethodPost {
		if len(body) > 0 && path != "/v1/service/status" {
			fmt.Println(string(body))
		}
	}
	return nil
}
