package routing

import (
	"context"
	"strconv"
	"strings"

	"github.com/muhomor/muhomor/internal/store"
)

// ShouldRouteRuGeoViaProxy — WhitelistRuRouting when exit is RU on WL network.
func ShouldRouteRuGeoViaProxy(ctx context.Context, st *store.Store, profileID int64) bool {
	wl, _ := st.GetKV(ctx, store.KeyActiveWhitelistRestricted)
	if wl != "true" {
		return false
	}
	pidStr, _ := st.GetKV(ctx, store.KeyVpnExitProbeProfileID)
	if pidStr == "" {
		return false
	}
	exitRU, _ := st.GetKV(ctx, store.KeyVpnExitIsRussia)
	return exitRU == "true" && pidStr == formatProfileID(profileID)
}

// ApplyRuGeoRules returns rules with RU direct overridden to PROXY when needed.
func ApplyRuGeoRules(ctx context.Context, st *store.Store, profileID int64, base []string) []string {
	if !ShouldRouteRuGeoViaProxy(ctx, st, profileID) {
		return base
	}
	out := make([]string, 0, len(base))
	for _, r := range base {
		if isRuDirectRule(r) {
			out = append(out, strings.Replace(r, ",DIRECT", ",PROXY", 1))
		} else {
			out = append(out, r)
		}
	}
	return out
}

func isRuDirectRule(r string) bool {
	r = strings.ToUpper(r)
	return strings.Contains(r, "GEOSITE,RU,DIRECT") ||
		strings.Contains(r, "GEOIP,RU,DIRECT") ||
		strings.Contains(r, "GEOSITE-CATEGORY-RU,DIRECT")
}

func formatProfileID(id int64) string {
	return strconv.FormatInt(id, 10)
}
