package configgen

import (
	"fmt"
	"strings"

	"github.com/muhomor/muhomor/internal/store"
)

// ChainSpec describes a relay chain (Phase 3).
type ChainSpec struct {
	Name   string
	Members []store.Profile // ordered: local -> ... -> exit
}

// BuildChainConfig builds config with relay proxy-group (mihomo dialer-proxy).
func BuildChainConfig(chain ChainSpec, opt BuildOptions, rules []string, rulesDir string) (yaml string, exitName string, err error) {
	if len(chain.Members) == 0 {
		return "", "", fmt.Errorf("empty chain")
	}
	if opt.Secret == "" {
		opt.Secret = randomSecret()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "mode: %s\n", opt.Mode)
	fmt.Fprintf(&b, "external-controller: %s\n", opt.ExternalController)
	fmt.Fprintf(&b, "secret: %q\n", opt.Secret)
	appendInboundSections(&b, opt.Inbound, opt.Tun)
	appendDNSSection(&b, opt.DNS)
	if rulesDir != "" {
		AppendRuleProviders(&b, rulesDir, store.RouteQuickRuBlockedAndAIProxy)
	}
	b.WriteString("\nproxies:\n")
	names := make([]string, 0, len(chain.Members))
	for _, m := range chain.Members {
		_, tag, err := appendProfileProxy(&b, m)
		if err != nil {
			return "", "", err
		}
		names = append(names, tag)
	}
	exitName = names[len(names)-1]
	b.WriteString("\nproxy-groups:\n")
	for i := 1; i < len(names); i++ {
		gname := fmt.Sprintf("chain-%d", i)
		fmt.Fprintf(&b, "  - name: %s\n    type: relay\n    proxies:\n      - %s\n      - %s\n", gname, names[i-1], names[i])
	}
	fmt.Fprintf(&b, "  - name: PROXY\n    type: select\n    proxies:\n      - %s\n", exitName)
	b.WriteString("\nrules:\n")
	if len(rules) == 0 {
		b.WriteString("  - MATCH,PROXY\n")
	} else {
		for _, r := range rules {
			fmt.Fprintf(&b, "  - %s\n", r)
		}
	}
	return b.String(), exitName, nil
}

func appendProfileProxy(b *strings.Builder, p store.Profile) (yaml string, tag string, err error) {
	tag, err = ProxyNameForProfile(p)
	if err != nil {
		return "", "", err
	}
	switch p.Type {
	case "trojan":
		t, err := ParseTrojanURI(p.URI)
		if err != nil {
			return "", "", err
		}
		writeTrojanProxy(b, tag, t)
	case "vless":
		v, err := ParseVLESSURI(p.URI)
		if err != nil {
			return "", "", err
		}
		writeVLESSProxy(b, tag, v)
	case "hysteria2", "hysteria":
		h, err := ParseHysteriaURI(p.URI)
		if err != nil {
			return "", "", err
		}
		writeHysteria2Proxy(b, tag, h)
	default:
		return "", "", fmt.Errorf("chain: unsupported type %q", p.Type)
	}
	return "", tag, nil
}
