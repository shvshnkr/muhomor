package subscription

// White Bolt WL catalog (russian-white-bolt curated subset for BS/WL networks).
// Black/open sources stay in DefaultLinks — not duplicated here.
const WhiteBoltWLGroupPrefix = "White Bolt WL: "

// WhiteBoltWLSource is one subscription feed for whitelist-only networks.
type WhiteBoltWLSource struct {
	Name string
	URL  string
}

// WhiteBoltWLSources from RUVIPIEN/russian-white-bolt sources/urls.txt (White only, no Black dupes).
var WhiteBoltWLSources = []WhiteBoltWLSource{
	{Name: "igareck VLESS Reality WL Mobile", URL: "https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/Vless-Reality-White-Lists-Rus-Mobile.txt"},
	{Name: "igareck VLESS Reality WL Mobile 2", URL: "https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/Vless-Reality-White-Lists-Rus-Mobile-2.txt"},
	{Name: "igareck WHITE-CIDR checked", URL: "https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/WHITE-CIDR-RU-checked.txt"},
	{Name: "SilentGhost Whitelist 1", URL: "https://raw.githubusercontent.com/SilentGhostCodes/WhiteListVpn/refs/heads/main/Whitelist.txt"},
	{Name: "SilentGhost Whitelist 2", URL: "https://raw.githubusercontent.com/SilentGhostCodes/WhiteListVpn/refs/heads/main/Whitelist%20%E2%84%962.txt"},
	{Name: "ByeWhiteLists 2", URL: "https://raw.githubusercontent.com/ByeWhiteLists/ByeWhiteLists2/refs/heads/main/ByeWhiteLists2.txt"},
	{Name: "ByWarm WL", URL: "https://gitverse.ru/api/repos/bywarm/rser/raw/branch/master/wl.txt"},
	{Name: "ByWarm Selected", URL: "https://gitverse.ru/api/repos/bywarm/rser/raw/branch/master/selected.txt"},
	{Name: "CID White List", URL: "https://gitverse.ru/api/repos/cid-uskoritel/cid-white/raw/branch/master/whitelist.txt"},
	{Name: "WhitePrime WL available", URL: "https://whiteprime.github.io/xraycheck/configs/white-list_available"},
}

// IsWhiteBoltWLGroup reports subscription groups created by BootstrapWhiteBoltWL.
func IsWhiteBoltWLGroup(name string) bool {
	return len(name) >= len(WhiteBoltWLGroupPrefix) && name[:len(WhiteBoltWLGroupPrefix)] == WhiteBoltWLGroupPrefix
}
