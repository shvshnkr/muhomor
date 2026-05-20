package subscription

import (
	"net/url"
	"strings"
)

// User-Agent profiles (parity fr.husi.group.SubscriptionFetchProfile + SubscriptionSourceKind).
const (
	// HappUserAgent is SubscriptionFetchProfile.HAPP in Dahusim.
	HappUserAgent = "happ/2.9.0"
	// BrowserUserAgent for providers that reject Happ/Go (e.g. some panels, mifa fallback).
	BrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	// HusiLikeUserAgent is SubscriptionFetchProfile.DEFAULT on GitHub hosts (fr.husi.ktx.USER_AGENT shape).
	HusiLikeUserAgent = "husi/1.0 (muhomor; sing-box mihomo)"
)

const (
	sourceGitHub = 0
	sourceWeb    = 1
)

var githubHosts = map[string]bool{
	"github.com":              true,
	"raw.githubusercontent.com": true,
	"gist.githubusercontent.com": true,
}

// SubscriptionSourceKind mirrors SubscriptionSourceKind.inferFromLink (Dahusim).
func SubscriptionSourceKind(link string) int {
	host := subscriptionHost(link)
	if githubHosts[host] {
		return sourceGitHub
	}
	return sourceWeb
}

func subscriptionHost(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// UAModeLabel is a short read-only label for UI (happ / browser / default / custom).
func UAModeLabel(ua string) string {
	ua = strings.TrimSpace(ua)
	switch ua {
	case HappUserAgent:
		return "happ"
	case BrowserUserAgent:
		return "browser"
	case HusiLikeUserAgent:
		return "default"
	case "":
		return "авто"
	default:
		if strings.HasPrefix(strings.ToLower(ua), "happ/") {
			return "happ"
		}
		if strings.Contains(ua, "Mozilla/") {
			return "browser"
		}
		if strings.HasPrefix(strings.ToLower(ua), "husi/") {
			return "default"
		}
		return "custom"
	}
}

// ProbeUserAgentCandidates returns UA strings to try (saved UA tried first by caller).
func ProbeUserAgentCandidates(link, savedUA string) []string {
	savedUA = strings.TrimSpace(savedUA)
	host := subscriptionHost(link)
	var order []string
	switch {
	case host == "mifa.world":
		// User: remember browser for mifa; probe Happ then browser (learn on success).
		order = []string{HappUserAgent, BrowserUserAgent, HusiLikeUserAgent}
	case SubscriptionSourceKind(link) == sourceGitHub:
		order = []string{HusiLikeUserAgent, HappUserAgent, BrowserUserAgent}
	default:
		order = []string{HappUserAgent, BrowserUserAgent, HusiLikeUserAgent}
	}
	out := make([]string, 0, len(order)+1)
	seen := map[string]bool{}
	add := func(ua string) {
		ua = strings.TrimSpace(ua)
		if ua == "" || seen[ua] {
			return
		}
		seen[ua] = true
		out = append(out, ua)
	}
	if savedUA != "" {
		add(savedUA)
	}
	for _, ua := range order {
		add(ua)
	}
	return out
}
