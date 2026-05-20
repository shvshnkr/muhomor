package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/muhomor/muhomor/internal/store"
)

// ExitProbe detects VPN exit country via HTTP through mihomo mixed proxy (VpnExitProbe parity).
type ExitProbe struct {
	ProxyPort int
	Store     *store.Store
}

const ipAPIJSON = "http://ip-api.com/json/?fields=status,countryCode"

var countryCodeRe = regexp.MustCompile(`"countryCode"\s*:\s*"([A-Za-z]{2})"`)

// ProbeAndStore returns true if exit is RU, false if not, nil if unknown.
func (e *ExitProbe) ProbeAndStore(ctx context.Context, profileID int64) *bool {
	if e.ProxyPort <= 0 {
		e.ProxyPort = 7890
	}
	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", e.ProxyPort))
	client := &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	code, ok := fetchCountry(ctx, client, ipAPIJSON)
	if !ok {
		return nil
	}
	isRU := code == "RU"
	_ = e.Store.SetKV(ctx, store.KeyVpnExitIsRussia, boolKV(isRU))
	_ = e.Store.SetKV(ctx, store.KeyVpnExitProbeProfileID, fmtInt(profileID))
	return &isRU
}

func fetchCountry(ctx context.Context, client *http.Client, rawURL string) (string, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", false
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", false
	}
	if m := countryCodeRe.FindSubmatch(body); len(m) == 2 {
		return string(m[1]), true
	}
	var j struct {
		CountryCode string `json:"countryCode"`
	}
	if json.Unmarshal(body, &j) == nil && j.CountryCode != "" {
		return j.CountryCode, true
	}
	return "", false
}

func boolKV(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func fmtInt(n int64) string {
	return fmt.Sprintf("%d", n)
}
