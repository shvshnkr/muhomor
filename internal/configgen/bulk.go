package configgen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/muhomor/muhomor/internal/store"
)

const (
	GroupPROXY     = "PROXY"
	GroupPROXYBulk = "PROXY_BULK"

	bulkLBInterval = 60
	bulkLBTimeout  = 8000
)

// BulkPlan describes whether bulk load-balance is active for this config build.
type BulkPlan struct {
	Active         bool
	FallbackReason string // disabled, legacy_mode, not_enough_legs
	MatchTarget    string // PROXY or PROXY_BULK
	PrimaryTag     string
	BulkTags       []string
}

// ResolveBulkPlan decides if PROXY_BULK should be used from rendered member tags.
func ResolveBulkPlan(set store.Settings, memberTags []string, primaryTag string) BulkPlan {
	plan := BulkPlan{MatchTarget: GroupPROXY, FallbackReason: "legacy_mode", PrimaryTag: primaryTag}
	if set.AggregationMode != store.AggregationModeFlowAggregate {
		return plan
	}
	if !set.BulkEnabled {
		plan.FallbackReason = "disabled"
		return plan
	}
	minLegs := set.BulkMinHealthyLegs
	if minLegs < 1 {
		minLegs = 2
	}
	if len(memberTags) < minLegs {
		plan.FallbackReason = "not_enough_legs"
		return plan
	}
	plan.Active = true
	plan.MatchTarget = GroupPROXYBulk
	plan.BulkTags = memberTags
	plan.FallbackReason = ""
	return plan
}

func capBulkTags(ordered []string, set store.Settings) []string {
	max := set.BulkMaxLegs
	if max <= 0 {
		switch set.MultipathPreset {
		case "low":
			max = 3
		case "high":
			max = 8
		default:
			max = 6
		}
	}
	if len(ordered) > max {
		return ordered[:max]
	}
	return ordered
}

func bulkProxyTag(p store.Profile) (string, error) {
	base, err := ProxyNameForProfile(p)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_p%d", base, p.ID), nil
}

// ProfileIDFromBulkTag parses profile id from mihomo tag suffix "_p{id}".
func ProfileIDFromBulkTag(tag string) (int64, bool) {
	i := strings.LastIndex(tag, "_p")
	if i < 0 || i+2 >= len(tag) {
		return 0, false
	}
	id, err := strconv.ParseInt(tag[i+2:], 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// RewriteMatchTarget replaces final MATCH rules to use target group.
func RewriteMatchTarget(rules []string, target string) []string {
	if target == "" {
		target = GroupPROXY
	}
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		if strings.HasPrefix(r, "MATCH,") {
			out = append(out, "MATCH,"+target)
		} else {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		out = []string{"MATCH," + target}
	}
	return out
}

// appendBulkProxyGroups writes PROXY select + optional PROXY_BULK load-balance.
func appendBulkProxyGroups(b *strings.Builder, plan BulkPlan, allTags []string, set store.Settings) {
	b.WriteString("  - name: ")
	b.WriteString(yamlQuote(GroupPROXY))
	b.WriteString("\n    type: select\n    proxies:\n")
	for _, t := range allTags {
		yamlProxyRef(b, t)
	}
	if plan.Active && len(plan.BulkTags) >= 2 {
		b.WriteString("  - name: ")
		b.WriteString(yamlQuote(GroupPROXYBulk))
		strategy := store.NormalizeBulkLBStrategy(set.BulkLBStrategy)
		fmt.Fprintf(b, "\n    type: load-balance\n    strategy: %s\n    url: \"http://www.gstatic.com/generate_204\"\n    interval: %d\n    timeout: %d\n    lazy: false\n    proxies:\n",
			strategy, bulkLBInterval, bulkLBTimeout)
		for _, t := range plan.BulkTags {
			yamlProxyRef(b, t)
		}
	}
}

// writeProfilesProxies writes all profile outbounds with unique tags.
func writeProfilesProxies(b *strings.Builder, profiles []store.Profile) (tags []string, primary string, err error) {
	seen := map[string]struct{}{}
	for _, p := range profiles {
		tag, e := bulkProxyTag(p)
		if e != nil {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		if err = writeProfileProxy(b, p, tag); err != nil {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	if len(tags) == 0 {
		return nil, "", fmt.Errorf("no valid proxies")
	}
	return tags, tags[0], nil
}
