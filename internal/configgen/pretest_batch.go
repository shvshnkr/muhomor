package configgen

import (
	"fmt"
	"strings"

	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/store"
)

// PretestGroupName is the url-test group used for batch delay (mihomo /group/.../delay).
const PretestGroupName = "MUHOMOR_PRETEST"

// BuildPretestBatch builds one mihomo config with url-test group for native parallel delay test.
func BuildPretestBatch(profiles []store.Profile, opt BuildOptions, testURL string, timeoutMs int) (yaml string, idToName map[int64]string, err error) {
	if opt.MixedPort == 0 {
		opt.MixedPort = 17890
	}
	if opt.ExternalController == "" {
		opt.ExternalController = paths.DefaultExternalController()
	}
	if opt.Secret == "" {
		opt.Secret = "pretest"
	}
	if opt.Mode == "" {
		opt.Mode = "rule"
	}
	if opt.LogLevel == "" {
		opt.LogLevel = "error"
	}
	if testURL == "" {
		testURL = "http://www.gstatic.com/generate_204"
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	idToName = make(map[int64]string)
	var b strings.Builder
	fmt.Fprintf(&b, "mixed-port: %d\n", opt.MixedPort)
	b.WriteString("allow-lan: false\n")
	fmt.Fprintf(&b, "mode: %s\n", opt.Mode)
	fmt.Fprintf(&b, "log-level: %s\n", opt.LogLevel)
	fmt.Fprintf(&b, "external-controller: %s\n", opt.ExternalController)
	fmt.Fprintf(&b, "secret: %q\n", opt.Secret)
	b.WriteString("\nproxies:\n")
	var names []string
	used := map[string]struct{}{}
	for _, p := range profiles {
		tag, err := pretestProxyTag(p)
		if err != nil {
			continue
		}
		if _, ok := used[tag]; ok {
			continue
		}
		if err := writeProfileProxy(&b, p, tag); err != nil {
			continue
		}
		used[tag] = struct{}{}
		names = append(names, tag)
		idToName[p.ID] = tag
	}
	if len(names) == 0 {
		return "", nil, fmt.Errorf("no valid proxies for pretest batch")
	}
	b.WriteString("\nproxy-groups:\n")
	fmt.Fprintf(&b, "  - name: %s\n", yamlQuote(PretestGroupName))
	b.WriteString("    type: url-test\n")
	fmt.Fprintf(&b, "    url: %s\n", yamlQuote(testURL))
	fmt.Fprintf(&b, "    timeout: %d\n", timeoutMs)
	b.WriteString("    lazy: false\n")
	b.WriteString("    interval: 86400\n")
	b.WriteString("    proxies:\n")
	for _, n := range names {
		yamlProxyRef(&b, n)
	}
	b.WriteString("\nrules:\n  - MATCH,")
	fmt.Fprintf(&b, "%s\n", PretestGroupName)
	return b.String(), idToName, nil
}

func pretestProxyTag(p store.Profile) (string, error) {
	typ := p.Type
	if typ == "" {
		typ = "vless"
	}
	switch typ {
	case "trojan":
		if _, err := ParseTrojanURI(p.URI); err != nil {
			return "", err
		}
	case "hysteria", "hysteria2":
		if _, err := ParseHysteriaURI(p.URI); err != nil {
			return "", err
		}
	case "vless":
		if _, err := ParseVLESSURI(p.URI); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported type %q", typ)
	}
	base, err := ProxyNameForProfile(p)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_p%d", base, p.ID), nil
}

func writeProfileProxy(b *strings.Builder, p store.Profile, tag string) error {
	typ := p.Type
	if typ == "" {
		typ = "vless"
	}
	switch typ {
	case "trojan":
		t, err := ParseTrojanURI(p.URI)
		if err != nil {
			return err
		}
		writeTrojanProxy(b, tag, t)
	case "hysteria", "hysteria2":
		h, err := ParseHysteriaURI(p.URI)
		if err != nil {
			return err
		}
		writeHysteria2Proxy(b, tag, h)
	case "vless":
		v, err := ParseVLESSURI(p.URI)
		if err != nil {
			return err
		}
		writeVLESSProxy(b, tag, v)
	default:
		return fmt.Errorf("unsupported type %q", typ)
	}
	return nil
}
