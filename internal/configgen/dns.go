package configgen

import (
	"fmt"
	"strings"
)

// DNSOptions basic DNS block for mihomo.
type DNSOptions struct {
	Enable   bool
	FakeIP   bool
	Remote   string // e.g. https://1.1.1.1/dns-query
	Direct   string
}

func appendDNSSection(b *strings.Builder, dns DNSOptions) {
	if !dns.Enable {
		return
	}
	b.WriteString("\ndns:\n")
	b.WriteString("  enable: true\n")
	if dns.FakeIP {
		b.WriteString("  enhanced-mode: fake-ip\n")
		b.WriteString("  fake-ip-range: 198.18.0.1/16\n")
	} else {
		b.WriteString("  enhanced-mode: redir-host\n")
	}
	if dns.Remote != "" {
		b.WriteString("  nameserver:\n")
		fmt.Fprintf(b, "    - %s\n", dns.Remote)
	}
	if dns.Direct != "" {
		b.WriteString("  default-nameserver:\n")
		fmt.Fprintf(b, "    - %s\n", dns.Direct)
	}
}
