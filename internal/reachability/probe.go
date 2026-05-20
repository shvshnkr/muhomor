package reachability

import (
	"context"
	"net"
	"net/http"
	"time"
)

// Result mirrors Dahusim NetworkReachability (subset).
type Result struct {
	GoogleReachable          bool
	DzenReachable            bool
	YaReachable              bool
	WhitelistSourceReachable bool
}

func (r Result) AnyReachable() bool {
	return r.GoogleReachable || r.DzenReachable || r.YaReachable || r.WhitelistSourceReachable
}

func (r Result) WhitelistOnly() bool {
	return !r.GoogleReachable && (r.DzenReachable || r.WhitelistSourceReachable)
}

const (
	googleURL = "http://www.google.com/generate_204"
	dzenURL   = "http://dzen.ru"
	yaURL     = "http://ya.ru"
)

// Probe checks open vs whitelist-style connectivity (fast parallel).
func Probe(ctx context.Context, fast bool) Result {
	timeout := 4 * time.Second
	if fast {
		timeout = 2 * time.Second
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: timeout}).DialContext,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	type probe struct {
		name string
		url  string
	}
	targets := []probe{
		{"google", googleURL},
		{"dzen", dzenURL},
		{"ya", yaURL},
	}
	ch := make(chan struct {
		name string
		ok   bool
	}, len(targets))
	for _, t := range targets {
		t := t
		go func() {
			ok := headOK(ctx, client, t.url)
			ch <- struct {
				name string
				ok   bool
			}{t.name, ok}
		}()
	}
	var r Result
	for range targets {
		select {
		case <-ctx.Done():
			return r
		case p := <-ch:
			switch p.name {
			case "google":
				r.GoogleReachable = p.ok
			case "dzen":
				r.DzenReachable = p.ok
			case "ya":
				r.YaReachable = p.ok
			}
		}
	}
	r.WhitelistSourceReachable = r.DzenReachable || r.YaReachable
	return r
}

func headOK(ctx context.Context, client *http.Client, rawURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return false
		}
		resp, err = client.Do(req)
		if err != nil {
			return false
		}
	}
	defer resp.Body.Close()
	return resp.StatusCode > 0 && resp.StatusCode < 500
}
