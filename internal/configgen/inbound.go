package configgen

import (
	"fmt"
	"strings"
)

// InboundOptions — mixed/socks/http + auth (ConfigBuilder inbound subset).
type InboundOptions struct {
	MixedPort   int
	SocksPort   int
	HTTPPort    int
	AllowLAN    bool
	BindAddress string // 127.0.0.1 or 0.0.0.0
	Username    string
	Password    string
}

func appendInboundSections(b *strings.Builder, in InboundOptions, tun TunOptions) {
	if in.MixedPort <= 0 {
		in.MixedPort = 7890
	}
	bind := in.BindAddress
	if bind == "" {
		if in.AllowLAN {
			bind = "0.0.0.0"
		} else {
			bind = "127.0.0.1"
		}
	}
	fmt.Fprintf(b, "mixed-port: %d\n", in.MixedPort)
	if in.SocksPort > 0 {
		fmt.Fprintf(b, "socks-port: %d\n", in.SocksPort)
	}
	if in.HTTPPort > 0 {
		fmt.Fprintf(b, "port: %d\n", in.HTTPPort)
	}
	fmt.Fprintf(b, "allow-lan: %t\n", in.AllowLAN)
	fmt.Fprintf(b, "bind-address: %q\n", bind)
	if in.Username != "" && in.Password != "" {
		b.WriteString("authentication:\n")
		fmt.Fprintf(b, "  - %q\n", in.Username+":"+in.Password)
	}
	if tun.Enable {
		appendTunSection(b, tun)
	}
}
