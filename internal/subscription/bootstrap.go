package subscription

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/store"
)

// DefaultLinks used when kit config/subscriptions.txt has no URLs (see subscriptions.example.txt).
var DefaultLinks = []string{
	"https://mifa.world/vless",
	store.SwordwareSubscriptionURL,
	"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/BLACK_VLESS_RUS_mobile.txt",
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
	links := SubscriptionLinksForBootstrap()
	u := &Updater{Store: st, Client: &http.Client{Timeout: 45 * time.Second}}
	for i, link := range links {
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

// SubscriptionLinksForBootstrap returns kit config/subscriptions.txt lines or built-in defaults.
func SubscriptionLinksForBootstrap() []string {
	if path, ok := paths.KitSubscriptionsFile(); ok {
		if links, err := loadSubscriptionLinksFile(path); err == nil && len(links) > 0 {
			return links
		}
	}
	return DefaultLinks
}

func loadSubscriptionLinksFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, sc.Err()
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
