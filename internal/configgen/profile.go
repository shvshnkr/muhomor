package configgen

import (
	"fmt"
	"strings"

	"github.com/muhomor/muhomor/internal/paths"
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
	yaml, proxyName, _, err = buildProfilesExt([]store.Profile{p}, p, opt, rules, rulesDir, quickProfile, store.Settings{AggregationMode: store.AggregationModeLegacy})
	return yaml, proxyName, err
}

func buildProfilesExt(profiles []store.Profile, primary store.Profile, opt BuildOptions, rules []string, rulesDir string, quickProfile int, set store.Settings) (yaml string, proxyName string, plan BulkPlan, err error) {
	if err = validatePrimaryProfile(primary); err != nil {
		return "", "", BulkPlan{}, err
	}
	return buildWithProfilesWriter(profiles, primary, set, opt, rules, rulesDir, quickProfile)
}

func validatePrimaryProfile(p store.Profile) error {
	switch p.Type {
	case "trojan":
		_, err := ParseTrojanURI(p.URI)
		return err
	case "hysteria2", "hysteria":
		_, err := ParseHysteriaURI(p.URI)
		return err
	case "vless", "":
		_, err := ParseVLESSURI(p.URI)
		return err
	default:
		return fmt.Errorf("unsupported profile type %q (see docs/PROTOCOL_GAP.md)", p.Type)
	}
}

// BuildFromProfileBulk builds YAML with optional bulk pool legs.
func BuildFromProfileBulk(primary store.Profile, poolLegs []store.Profile, opt BuildOptions, rules []string, rulesDir string, quickProfile int, set store.Settings) (yaml string, proxyName string, plan BulkPlan, err error) {
	all := dedupeProfiles(append([]store.Profile{primary}, poolLegs...))
	return buildProfilesExt(all, primary, opt, rules, rulesDir, quickProfile, set)
}

func dedupeProfiles(in []store.Profile) []store.Profile {
	seen := map[int64]struct{}{}
	var out []store.Profile
	for _, p := range in {
		if p.ID == 0 {
			continue
		}
		if _, ok := seen[p.ID]; ok {
			continue
		}
		seen[p.ID] = struct{}{}
		out = append(out, p)
	}
	return out
}

func buildWithProfilesWriter(profiles []store.Profile, primary store.Profile, set store.Settings, opt BuildOptions, rules []string, rulesDir string, quickProfile int) (string, string, BulkPlan, error) {
	if opt.MixedPort == 0 {
		opt.MixedPort = 7890
	}
	if opt.ExternalController == "" {
		opt.ExternalController = paths.DefaultExternalController()
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
	allProfiles := dedupeProfiles(profiles)
	if len(allProfiles) == 0 {
		allProfiles = []store.Profile{primary}
	}
	tags, primaryTag, err := writeProfilesProxies(&b, allProfiles)
	if err != nil {
		return "", "", BulkPlan{}, err
	}
	ordered := orderPrimaryFirst(tags, primaryTag)
	bulkTags := capBulkTags(ordered, set)
	plan := ResolveBulkPlan(set, bulkTags, primaryTag)
	b.WriteString("\nproxy-groups:\n")
	appendBulkProxyGroups(&b, plan, ordered, set)
	proxyName := primaryTag
	effectiveRules := rules
	if len(effectiveRules) == 0 {
		appendRuDirectRulesWithMatch(&b, plan.MatchTarget)
	} else {
		effectiveRules = RewriteMatchTarget(effectiveRules, plan.MatchTarget)
		b.WriteString("\nrules:\n")
		for _, r := range effectiveRules {
			fmt.Fprintf(&b, "  - %s\n", r)
		}
	}
	return b.String(), proxyName, plan, nil
}

func orderPrimaryFirst(tags []string, primary string) []string {
	if primary == "" || len(tags) <= 1 {
		return tags
	}
	out := []string{primary}
	for _, t := range tags {
		if t != primary {
			out = append(out, t)
		}
	}
	return out
}

func appendRuDirectRulesWithMatch(b *strings.Builder, matchTarget string) {
	b.WriteString("\nrules:\n")
	b.WriteString("  - GEOIP,ru,DIRECT\n")
	b.WriteString("  - GEOIP,private,DIRECT\n")
	fmt.Fprintf(b, "  - MATCH,%s\n", matchTarget)
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
