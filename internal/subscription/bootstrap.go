package subscription

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/muhomor/muhomor/internal/store"
)

// DefaultLinks from DefaultUserBootstrap.defaultSubscriptionLinks (Dahusim).
var DefaultLinks = []string{
	"https://mifa.world/vless",
	"https://mifa.world/hysteria",
	store.SwordwareSubscriptionURL,
	"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/BLACK_VLESS_RUS_mobile.txt",
	"https://gist.githubusercontent.com/flaafix/c79a81037d15163360571c7a7331b153/raw/AetrisVPN.txt",
	"https://raw.githubusercontent.com/nzea243/ikoV31tud_vpn/refs/heads/main/tri_228.txt",
	"https://raw.githubusercontent.com/HikaruApps/WhiteLattice/refs/heads/main/subscriptions/config.txt",
	"https://raw.githubusercontent.com/SilentGhostCodes/WhiteListVpn/refs/heads/main/BlackList.txt",
	"https://wlrus.lol/confs/blackl.txt",
}

// Bootstrap imports default subscription links when groups are empty.
func Bootstrap(ctx context.Context, st *store.Store) error {
	groups, err := st.ListGroups(ctx)
	if err != nil {
		return err
	}
	if len(groups) > 0 {
		return nil
	}
	u := &Updater{Store: st, Client: &http.Client{Timeout: 45 * time.Second}}
	for i, link := range DefaultLinks {
		gid, err := st.CreateGroup(ctx, fmt.Sprintf("Quick Subscription %d", i+1), link, store.GroupKindSubscription)
		if err != nil {
			continue
		}
		_ = u.FetchGroup(ctx, gid, link)
	}
	qp, _ := st.RouteQuickProfile(ctx)
	if qp == store.RouteQuickManual {
		_ = st.SetRouteQuickProfile(ctx, store.RouteQuickRuDirectOnly)
	}
	return nil
}

// HasWhiteBoltWLGroups returns true if any White Bolt WL subscription group exists.
func HasWhiteBoltWLGroups(ctx context.Context, st *store.Store) (bool, error) {
	groups, err := st.ListGroups(ctx)
	if err != nil {
		return false, err
	}
	for _, g := range groups {
		if IsWhiteBoltWLGroup(g.Name) {
			return true, nil
		}
	}
	return false, nil
}

// BootstrapWhiteBoltWL creates WL subscription groups from curated white-bolt sources.
// Idempotent: skips if any White Bolt WL group already exists.
func BootstrapWhiteBoltWL(ctx context.Context, st *store.Store) error {
	ok, err := HasWhiteBoltWLGroups(ctx, st)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	u := &Updater{Store: st, Client: &http.Client{Timeout: 45 * time.Second}}
	for _, src := range WhiteBoltWLSources {
		name := WhiteBoltWLGroupPrefix + src.Name
		gid, err := st.CreateGroup(ctx, name, src.URL, store.GroupKindSubscription)
		if err != nil {
			continue
		}
		_, _ = u.RefreshGroup(ctx, gid)
	}
	return nil
}

// ListWhiteBoltWLGroupIDs returns subscription group ids for White Bolt WL feeds.
func ListWhiteBoltWLGroupIDs(ctx context.Context, st *store.Store) ([]int64, error) {
	groups, err := st.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for _, g := range groups {
		if g.Kind == store.GroupKindSubscription && g.SubscriptionLink != "" && IsWhiteBoltWLGroup(g.Name) {
			ids = append(ids, g.ID)
		}
	}
	return ids, nil
}

// ListOpenNetworkSubscriptionGroupIDs returns Black/open subscription groups (not WL catalog).
func ListOpenNetworkSubscriptionGroupIDs(ctx context.Context, st *store.Store) ([]int64, error) {
	groups, err := st.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for _, g := range groups {
		if g.Kind != store.GroupKindSubscription || g.SubscriptionLink == "" {
			continue
		}
		if IsWhiteBoltWLGroup(g.Name) {
			continue
		}
		if strings.TrimSpace(g.Name) == store.BuiltinWLGroupName {
			continue
		}
		ids = append(ids, g.ID)
	}
	return ids, nil
}
