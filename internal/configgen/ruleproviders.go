package configgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/muhomor/muhomor/internal/store"
)

// Meta-rules-dat clash-compatible lists (mihomo classical provider).
const (
	metaRulesBase = "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo"
)

// AppendRuleProviders adds rule-providers block for RU blocked + AI quick profile.
func AppendRuleProviders(b *strings.Builder, rulesDir string, quickProfile int) {
	if quickProfile != store.RouteQuickRuBlockedAndAIProxy {
		return
	}
	_ = os.MkdirAll(rulesDir, 0o755)
	providers := []struct {
		name string
		url  string
	}{
		{"geosite-ru-blocked", metaRulesBase + "/geosite/category-ru-blocked.yaml"},
		{"geosite-openai", metaRulesBase + "/geosite/openai.yaml"},
		{"geosite-anthropic", metaRulesBase + "/geosite/anthropic.yaml"},
		{"geosite-google-gemini", metaRulesBase + "/geosite/google-gemini.yaml"},
	}
	b.WriteString("\nrule-providers:\n")
	for _, p := range providers {
		local := filepath.Join(rulesDir, p.name+".yaml")
		fmt.Fprintf(b, "  %s:\n", p.name)
		b.WriteString("    type: http\n    behavior: classical\n")
		fmt.Fprintf(b, "    url: %q\n", p.url)
		fmt.Fprintf(b, "    path: %q\n", local)
		b.WriteString("    interval: 86400\n")
	}
}
