package configgen

import "strings"

// TunOptions — Phase 3 parity with ConfigBuilder TUN section (subset).
type TunOptions struct {
	Enable   bool
	Stack    string // system / gvisor
	DNSHijack bool
	MTU      int
}

func appendTunSection(b *strings.Builder, tun TunOptions) {
	b.WriteString("\ntun:\n")
	b.WriteString("  enable: true\n")
	if tun.Stack != "" {
		b.WriteString("  stack: " + tun.Stack + "\n")
	} else {
		b.WriteString("  stack: system\n")
	}
	if tun.DNSHijack {
		b.WriteString("  dns-hijack:\n    - any:53\n")
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
