package configgen

import (
	"fmt"
	"strings"

	"github.com/muhomor/muhomor/internal/paths"
)

// BuildPickerBootstrap writes minimal mihomo config so picker subprocess can start before profile reload.
func BuildPickerBootstrap(opt BuildOptions) string {
	if opt.MixedPort == 0 {
		opt.MixedPort = 2182
	}
	if opt.ExternalController == "" {
		opt.ExternalController = paths.DefaultExternalController()
	}
	if opt.Secret == "" {
		opt.Secret = "picker"
	}
	if opt.Mode == "" {
		opt.Mode = "rule"
	}
	if opt.LogLevel == "" {
		opt.LogLevel = "error"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "mixed-port: %d\n", opt.MixedPort)
	b.WriteString("allow-lan: false\n")
	fmt.Fprintf(&b, "mode: %s\n", opt.Mode)
	fmt.Fprintf(&b, "log-level: %s\n", opt.LogLevel)
	fmt.Fprintf(&b, "external-controller: %s\n", opt.ExternalController)
	fmt.Fprintf(&b, "secret: %q\n", opt.Secret)
	b.WriteString("\nproxies: []\n\nrules:\n  - MATCH,DIRECT\n")
	return b.String()
}
