package configgen

import (
	"fmt"
	"strings"

	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/store"
)

// BL group names for multi-URL suite (mihomo /group/{name}/delay).
const (
	PretestBLGroupTG = "MUHOMOR_BL_TG"
	PretestBLGroupWA = "MUHOMOR_BL_WA"
)

// BuildPretestBLBatch builds one url-test group for a single BL URL.
func BuildPretestBLBatch(profiles []store.Profile, opt BuildOptions, testURL string, timeoutMs int, groupName string) (yaml string, idToName map[int64]string, err error) {
	return buildPretestURLGroup(profiles, opt, testURL, timeoutMs, groupName)
}

// BuildPretestBLMulti builds several url-test groups (one URL each) in one config.
func BuildPretestBLMulti(profiles []store.Profile, opt BuildOptions, urls []string, timeoutMs int) (yaml string, groups []string, idToName map[int64]string, err error) {
	if len(urls) == 0 {
		return "", nil, nil, fmt.Errorf("no BL test URLs")
	}
	idToName = make(map[int64]string)
	var b strings.Builder
	for i, u := range urls {
		gname := blGroupNameForURL(u, i)
		part, names, e := buildPretestURLGroup(profiles, opt, u, timeoutMs, gname)
		if e != nil {
			continue
		}
		for id, name := range names {
			idToName[id] = name
		}
		if i == 0 {
			b.WriteString(part)
			groups = append(groups, gname)
			continue
		}
		// Append proxy-groups section entries from subsequent configs.
		idx := strings.Index(part, "\nproxy-groups:\n")
		if idx >= 0 {
			b.WriteString(part[idx+len("\nproxy-groups:"):])
			groups = append(groups, gname)
		}
	}
	if len(groups) == 0 {
		return "", nil, nil, fmt.Errorf("no valid BL groups")
	}
	return b.String(), groups, idToName, nil
}

func blGroupNameForURL(u string, idx int) string {
	lu := strings.ToLower(u)
	switch {
	case strings.Contains(lu, "telegram"):
		return PretestBLGroupTG
	case strings.Contains(lu, "whatsapp"):
		return PretestBLGroupWA
	default:
		return fmt.Sprintf("MUHOMOR_BL_%d", idx)
	}
}

func buildPretestURLGroup(profiles []store.Profile, opt BuildOptions, testURL string, timeoutMs int, groupName string) (yaml string, idToName map[int64]string, err error) {
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
		return "", nil, fmt.Errorf("empty test URL")
	}
	if timeoutMs <= 0 {
		timeoutMs = 8000
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
		return "", nil, fmt.Errorf("no valid proxies for BL pretest")
	}
	b.WriteString("\nproxy-groups:\n")
	fmt.Fprintf(&b, "  - name: %s\n", yamlQuote(groupName))
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
	fmt.Fprintf(&b, "%s\n", groupName)
	return b.String(), idToName, nil
}
