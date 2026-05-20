package subscription

import "testing"

func TestProbeUserAgentCandidates_web_defaultHapp(t *testing.T) {
	c := ProbeUserAgentCandidates("https://sub.example.com/api", "")
	if len(c) < 2 || c[0] != HappUserAgent {
		t.Fatalf("web should try happ first: %v", c)
	}
}

func TestProbeUserAgentCandidates_github_defaultHusi(t *testing.T) {
	c := ProbeUserAgentCandidates("https://raw.githubusercontent.com/x/y/z.txt", "")
	if c[0] != HusiLikeUserAgent {
		t.Fatalf("github should try husi-like first: %v", c)
	}
}

func TestProbeUserAgentCandidates_savedFirst(t *testing.T) {
	c := ProbeUserAgentCandidates("https://mifa.world/vless", BrowserUserAgent)
	if c[0] != BrowserUserAgent {
		t.Fatalf("saved first: %v", c)
	}
}

func TestUAModeLabel(t *testing.T) {
	if UAModeLabel(HappUserAgent) != "happ" {
		t.Fatal()
	}
	if UAModeLabel(BrowserUserAgent) != "browser" {
		t.Fatal()
	}
}
