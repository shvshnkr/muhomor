package configgen

import "github.com/muhomor/muhomor/internal/store"

// OptionsFromSettings maps persisted settings to BuildOptions.
func OptionsFromSettings(s store.Settings, base BuildOptions) BuildOptions {
	if base.Secret == "" {
		base = DefaultBuildOptions()
	}
	base.MixedPort = s.MixedPort
	base.Inbound = InboundOptions{
		MixedPort: s.MixedPort,
		SocksPort: s.SocksPort,
		HTTPPort:  s.HTTPPort,
		AllowLAN:  s.AllowLAN,
		Username:  s.InboundUser,
		Password:  s.InboundPassword,
	}
	if s.TunEnable {
		base.Tun = DefaultTunOptions(s.TunStack, s.TunMTU, s.RouteQuickProfile == store.RouteQuickWGOverWLTunnel)
	} else {
		base.Tun = TunOptions{Enable: false}
	}
	base.DNS = DNSOptions{
		Enable: true,
		FakeIP: s.DNSFakeIP,
		Remote: "https://1.1.1.1/dns-query",
		Direct: "223.5.5.5",
	}
	return base
}
