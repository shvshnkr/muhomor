package configgen

import (
	"strings"
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestAppendTunSection_wgOverWL_strictRouteAndSplit(t *testing.T) {
	var b strings.Builder
	appendTunSection(&b, DefaultTunOptions("system", 9000, true))
	out := b.String()
	for _, want := range []string{
		"auto-route: true",
		"strict-route: true",
		"auto-detect-interface: true",
		"stack: mixed",
		"0.0.0.0/1",
		"tcp://any:53",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in tun section:\n%s", want, out)
		}
	}
}

func TestOptionsFromSettings_vpnEnablesTunCapture(t *testing.T) {
	opt := OptionsFromSettings(store.Settings{
		TunEnable:           true,
		TunStack:            "system",
		TunDNSHijack:        true,
		TunMTU:              9000,
		RouteQuickProfile:   store.RouteQuickWGOverWLTunnel,
	}, DefaultBuildOptions())
	if !opt.Tun.StrictRoute || !opt.Tun.RouteAddressSplit {
		t.Fatalf("expected strict split tun for WG preset: %+v", opt.Tun)
	}
}
