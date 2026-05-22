package subscription

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// AssetUpdater refreshes geo/rule files (RouteAssetUpdater subset, Phase 3).
type AssetUpdater struct {
	RulesDir string
	Client   *http.Client
}

var assetURLs = []struct{ name, url string }{
	{"geosite-ru-blocked", "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/category-ru-blocked.yaml"},
	{"geosite-openai", "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/openai.yaml"},
	{"geosite-anthropic", "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/anthropic.yaml"},
}

// UpdateIfDue downloads rule-provider YAML files. Returns error if every fetch failed.
func (a *AssetUpdater) UpdateIfDue(ctx context.Context) error {
	if a.RulesDir == "" {
		return nil
	}
	if a.Client == nil {
		a.Client = &http.Client{Timeout: 60 * time.Second}
	}
	_ = os.MkdirAll(a.RulesDir, 0o755)
	var okCount int
	var lastErr error
	for _, item := range assetURLs {
		if err := a.fetchOne(ctx, item.name, item.url); err != nil {
			lastErr = err
			continue
		}
		okCount++
	}
	if okCount == 0 && lastErr != nil {
		return fmt.Errorf("asset update: all fetches failed: %w", lastErr)
	}
	if okCount == 0 {
		return fmt.Errorf("asset update: no rule files downloaded")
	}
	marker := filepath.Join(a.RulesDir, ".last-asset-update")
	return os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)), 0o644)
}

func (a *AssetUpdater) fetchOne(ctx context.Context, name, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(a.RulesDir, name+".yaml"), body, 0o644)
}
