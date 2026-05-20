package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/paths"
)

// RunCtlChain POST /v1/service/chain with profile ids.
func RunCtlChain(layout paths.Layout, idsCSV string) error {
	var ids []int64
	for _, p := range strings.Split(idsCSV, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}
	body, _ := json.Marshal(map[string]any{"ids": ids})
	dial := apiclient.DialConfig{SocketPath: layout.SocketPath()}
	tr := &http.Transport{DialContext: dial.DialContext}
	client := &http.Client{Timeout: 120 * time.Second, Transport: tr}
	req, err := http.NewRequest(http.MethodPost, "http://localhost/v1/service/chain", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("daemon not running: %w", err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(out)))
	}
	fmt.Println(string(out))
	return nil
}
