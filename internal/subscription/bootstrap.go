package subscription

import (
	"context"
	"fmt"
	"net/http"
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
