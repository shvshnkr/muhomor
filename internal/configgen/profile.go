package configgen

import (
	"fmt"
	"strings"

	"github.com/muhomor/muhomor/internal/store"
)

// BuildFromProfile builds mihomo YAML for a stored profile (vless/trojan).
func BuildFromProfile(p store.Profile, opt BuildOptions, rules []string) (yaml string, proxyName string, err error) {
	return BuildFromProfileExt(p, opt, rules, "")
}

// BuildFromProfileExt adds optional rulesDir for rule-providers cache.
// BuildFromProfileExt builds YAML with rule-providers dir and quick profile for providers.
func BuildFromProfileExt(p store.Profile, opt BuildOptions, rules []string, rulesDir string) (yaml string, proxyName string, err error) {
	qp := store.RouteQuickRuDirectOnly
	return buildProfileExt(p, opt, rules, rulesDir, qp)
}

// BuildFromProfileExtQP allows passing quick profile id.
func BuildFromProfileExtQP(p store.Profile, opt BuildOptions, rules []string, rulesDir string, quickProfile int) (yaml string, proxyName string, err error) {
	return buildProfileExt(p, opt, rules, rulesDir, quickProfile)
}

func buildProfileExt(p store.Profile, opt BuildOptions, rules []string, rulesDir string, quickProfile int) (yaml string, proxyName string, err error) {
	switch p.Type {
	case "trojan":
		t, err := ParseTrojanURI(p.URI)
		if err != nil {
			return "", "", err
		}
		if t.Name == "" {
			t.Name = p.Name
		}
		return buildWithProxyWriter(t.Name, opt, rules, rulesDir, quickProfile, func(b *strings.Builder, name string) {
			writeTrojanProxy(b, name, t)
		})
	case "hysteria2", "hysteria":
		h, err := ParseHysteriaURI(p.URI)
		if err != nil {
			return "", "", err
		}
		if h.Name == "" {
			h.Name = p.Name
		}
		return buildWithProxyWriter(h.Name, opt, rules, rulesDir, quickProfile, func(b *strings.Builder, name string) {
			writeHysteria2Proxy(b, name, h)
		})
	case "vless", "":
		v, err := ParseVLESSURI(p.URI)
		if err != nil {
			return "", "", err
		}
		if v.Name == "" {
			v.Name = p.Name
		}
		return buildWithProxyWriter(v.Name, opt, rules, rulesDir, quickProfile, func(b *strings.Builder, name string) {
			writeVLESSProxy(b, name, v)
		})
	default:
		return "", "", fmt.Errorf("unsupported profile type %q (see docs/PROTOCOL_GAP.md)", p.Type)
	}
}

func buildWithProxyWriter(name string, opt BuildOptions, rules []string, rulesDir string, quickProfile int, write func(*strings.Builder, string)) (string, string, error) {
	if opt.MixedPort == 0 {
		opt.MixedPort = 7890
	}
	if opt.ExternalController == "" {
		opt.ExternalController = "127.0.0.1:9090"
	}
	if opt.Secret == "" {
		opt.Secret = randomSecret()
	}
	if opt.Mode == "" {
		opt.Mode = "rule"
	}
	if opt.LogLevel == "" {
		opt.LogLevel = "info"
	}
	proxyName := sanitizeName(name)
	var b strings.Builder
	in := opt.Inbound
	if in.MixedPort == 0 {
		in.MixedPort = opt.MixedPort
	}
	if in.MixedPort == 0 {
		in.MixedPort = 7890
	}
	fmt.Fprintf(&b, "mode: %s\n", opt.Mode)
	fmt.Fprintf(&b, "log-level: %s\n", opt.LogLevel)
	fmt.Fprintf(&b, "external-controller: %s\n", opt.ExternalController)
	fmt.Fprintf(&b, "secret: %q\n", opt.Secret)
	appendInboundSections(&b, in, opt.Tun)
	appendDNSSection(&b, opt.DNS)
	if rulesDir != "" {
		AppendRuleProviders(&b, rulesDir, quickProfile)
	}
	b.WriteString("\nproxies:\n")
	write(&b, proxyName)
	b.WriteString("\nproxy-groups:\n")
	fmt.Fprintf(&b, "  - name: PROXY\n    type: select\n    proxies:\n      - %s\n", proxyName)
	if len(rules) == 0 {
		appendRuDirectRules(&b)
	} else {
		b.WriteString("\nrules:\n")
		for _, r := range rules {
			fmt.Fprintf(&b, "  - %s\n", r)
		}
	}
	return b.String(), proxyName, nil
}

// ProxyNameForProfile returns mihomo proxy tag for a profile.
func ProxyNameForProfile(p store.Profile) (string, error) {
	switch p.Type {
	case "trojan":
		t, err := ParseTrojanURI(p.URI)
		if err != nil {
			return "", err
		}
		if t.Name == "" {
			t.Name = p.Name
		}
		return sanitizeName(t.Name), nil
	case "hysteria2", "hysteria":
		h, err := ParseHysteriaURI(p.URI)
		if err != nil {
			return "", err
		}
		if h.Name == "" {
			h.Name = p.Name
		}
		return sanitizeName(h.Name), nil
	case "vless", "":
		v, err := ParseVLESSURI(p.URI)
		if err != nil {
			return "", err
		}
		if v.Name == "" {
			v.Name = p.Name
		}
		return sanitizeName(v.Name), nil
	default:
		return "", fmt.Errorf("unsupported type %q", p.Type)
	}
}
