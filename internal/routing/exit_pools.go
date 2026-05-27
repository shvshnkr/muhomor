package routing

import (
	"github.com/muhomor/muhomor/internal/profileclass"
	"github.com/muhomor/muhomor/internal/store"
)

// ProxyGroupRuExit is the future mihomo selector group for WL YouTube / RU-exit routing.
const ProxyGroupRuExit = "PROXY_RU_EXIT"

// RuExitEligibleForWL reports profiles that may be duplicated into PROXY_RU_EXIT (not implemented).
func RuExitEligibleForWL(p store.Profile) bool {
	return profileclass.IsRuExitMarked(p)
}

// TODO(wl_ru_exit_youtube): when feature-flag wl_ru_exit_youtube is on, duplicate ru_exit
// proxies into a second selector group and add GEOSITE,youtube,PROXY_RU_EXIT in configgen.
// BL-exit filter on pre-connect is separate from ExitProbe post-connect telemetry (vpn_exit_is_russia).
