package routing

import "github.com/muhomor/muhomor/internal/store"

// QuickProfileRuleLines returns mihomo rules for route quick profile.
// Invariant: blocked + AI PROXY rules must appear before RU DIRECT.
func QuickProfileRuleLines(profile int) []string {
	switch profile {
	case store.RouteQuickWGOverWLTunnel:
		// WL emergency preset: avoid DIRECT; UDP (WireGuard handshake) must use PROXY-capable outbound.
		return []string{
			"NETWORK,udp,PROXY",
			"GEOIP,private,PROXY",
			"MATCH,PROXY",
		}
	case store.RouteQuickRuBlockedAndAIProxy:
		return []string{
			"RULE-SET,geosite-ru-blocked,PROXY",
			"RULE-SET,geosite-ru-blocked-all,PROXY",
			"RULE-SET,geoip-ru-blocked,PROXY",
			"RULE-SET,geoip-ru-blocked-community,PROXY",
			"RULE-SET,geosite-openai,PROXY",
			"RULE-SET,geosite-anthropic,PROXY",
			"RULE-SET,geosite-google-gemini,PROXY",
			"RULE-SET,geosite-xai,PROXY",
			"GEOIP,ru,DIRECT",
			"GEOIP,private,DIRECT",
			"MATCH,PROXY",
		}
	case store.RouteQuickRuDirectOnly:
		fallthrough
	default:
		return []string{
			"GEOIP,ru,DIRECT",
			"GEOIP,private,DIRECT",
			"MATCH,PROXY",
		}
	}
}

// BlockedAIProviderNote: .srs URLs from sing-box need conversion for mihomo (see docs/RULE_PROVIDERS_RND.md).
// RULE-SET lines in QuickProfileRuleLines apply once providers are mirrored in config dir.

