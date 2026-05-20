package configgen

import "strings"

// TunOptions — Phase 3 parity with ConfigBuilder TUN section (Dahusim desktop subset).
type TunOptions struct {
	Enable              bool
	Stack               string // system / gvisor / mixed
	DNSHijack           bool
	MTU                 int
	AutoRoute           bool
	AutoDetectInterface bool
	StrictRoute         bool
	// RouteAddressSplit uses 0.0.0.0/1 + 128.0.0.0/1 instead of default route (Windows dual-default fix).
	RouteAddressSplit bool
}

func appendTunSection(b *strings.Builder, tun TunOptions) {
	b.WriteString("\ntun:\n")
	b.WriteString("  enable: true\n")
	stack := tun.Stack
	if stack == "" {
		stack = "mixed"
	}
	b.WriteString("  stack: " + stack + "\n")
	if tun.AutoRoute {
		b.WriteString("  auto-route: true\n")
	}
	if tun.AutoDetectInterface {
		b.WriteString("  auto-detect-interface: true\n")
	}
	if tun.StrictRoute {
		b.WriteString("  strict-route: true\n")
	}
	if tun.RouteAddressSplit {
		b.WriteString("  route-address:\n")
		b.WriteString("    - 0.0.0.0/1\n")
		b.WriteString("    - 128.0.0.0/1\n")
		b.WriteString("    - \"::/1\"\n")
		b.WriteString("    - \"8000::/1\"\n")
	}
	if tun.DNSHijack {
		b.WriteString("  dns-hijack:\n")
		b.WriteString("    - any:53\n")
		b.WriteString("    - tcp://any:53\n")
	}
	if tun.MTU > 0 {
		b.WriteString("  mtu: ")
		b.WriteString(strings.TrimSpace(strings.Replace(formatInt(tun.MTU), " ", "", -1)))
		b.WriteByte('\n')
	}
}

func formatInt(n int) string {
	if n <= 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// DefaultTunOptions returns TUN options for VPN mode (strict capture for WG-over-tunnel preset).
func DefaultTunOptions(stack string, mtu int, wgOverWL bool) TunOptions {
	if stack == "" || (wgOverWL && stack == "system") {
		stack = "mixed"
	}
	if mtu <= 0 {
		mtu = 9000
	}
	return TunOptions{
		Enable:              true,
		Stack:               stack,
		DNSHijack:           true,
		MTU:                 mtu,
		AutoRoute:           true,
		AutoDetectInterface: true,
		StrictRoute:         true,
		RouteAddressSplit:   wgOverWL,
	}
}
