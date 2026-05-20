package configgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// BuildOptions controls generated mihomo YAML.
type BuildOptions struct {
	MixedPort          int
	ExternalController string
	Secret             string
	Mode               string // rule / global / direct
	LogLevel           string
	Tun                TunOptions
	Inbound            InboundOptions
	DNS                DNSOptions
}

// DefaultBuildOptions returns sensible Linux desktop defaults.
func DefaultBuildOptions() BuildOptions {
	secret := randomSecret()
	return BuildOptions{
		MixedPort:          7890,
		ExternalController: "127.0.0.1:9090",
		Secret:             secret,
		Mode:               "rule",
		LogLevel:           "info",
	}
}

func randomSecret() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// BuildFromVLESS builds minimal mihomo config for one VLESS outbound.
func BuildFromVLESS(p VLESSProfile, opt BuildOptions) (yaml string, proxyName string, err error) {
	return BuildFromVLESSWithRules(p, opt, nil)
}

// BuildFromVLESSWithRules builds config with explicit rule lines (routing quick profile).
func BuildFromVLESSWithRules(p VLESSProfile, opt BuildOptions, rules []string) (yaml string, proxyName string, err error) {
	if opt.MixedPort == 0 {
		opt.MixedPort = 7890
	}
	if opt.ExternalController == "" {
		opt.ExternalController = "127.0.0.1:9090"
	}
	if opt.Secret == "" {
		opt.Secret = randomSecret()
	}
	if opt.Mode == "" {
		opt.Mode = "rule"
	}
	if opt.LogLevel == "" {
		opt.LogLevel = "info"
	}
	proxyName = sanitizeName(p.Name)
	var b strings.Builder
	fmt.Fprintf(&b, "mixed-port: %d\n", opt.MixedPort)
	fmt.Fprintf(&b, "allow-lan: false\n")
	fmt.Fprintf(&b, "mode: %s\n", opt.Mode)
	fmt.Fprintf(&b, "log-level: %s\n", opt.LogLevel)
	fmt.Fprintf(&b, "external-controller: %s\n", opt.ExternalController)
	fmt.Fprintf(&b, "secret: %q\n", opt.Secret)
	b.WriteString("\nproxies:\n")
	writeVLESSProxy(&b, proxyName, p)
	b.WriteString("\nproxy-groups:\n")
	fmt.Fprintf(&b, "  - name: PROXY\n    type: select\n    proxies:\n      - %s\n", proxyName)
	if len(rules) == 0 {
		appendRuDirectRules(&b)
	} else {
		b.WriteString("\nrules:\n")
		for _, r := range rules {
			fmt.Fprintf(&b, "  - %s\n", r)
		}
	}
	return b.String(), proxyName, nil
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "node-1"
	}
	repl := strings.NewReplacer(" ", "_", ":", "_", "/", "_", "\\", "_")
	return repl.Replace(name)
}

func writeVLESSProxy(b *strings.Builder, name string, p VLESSProfile) {
	fmt.Fprintf(b, "  - name: %s\n", name)
	b.WriteString("    type: vless\n")
	fmt.Fprintf(b, "    server: %s\n", p.Server)
	fmt.Fprintf(b, "    port: %d\n", p.Port)
	fmt.Fprintf(b, "    uuid: %s\n", p.UUID)
	if p.Flow != "" {
		fmt.Fprintf(b, "    flow: %s\n", p.Flow)
	}
	network := strings.ToLower(p.Network)
	if network == "" {
		network = "tcp"
	}
	if network != "tcp" {
		fmt.Fprintf(b, "    network: %s\n", network)
	}
	sec := strings.ToLower(p.Security)
	switch sec {
	case "tls", "reality":
		b.WriteString("    tls: true\n")
		if p.SNI != "" {
			fmt.Fprintf(b, "    servername: %s\n", p.SNI)
		}
		if p.ALPN != "" {
			fmt.Fprintf(b, "    alpn:\n")
			for _, a := range strings.Split(p.ALPN, ",") {
				a = strings.TrimSpace(a)
				if a != "" {
					fmt.Fprintf(b, "      - %s\n", a)
				}
			}
		}
		if p.FP != "" {
			fmt.Fprintf(b, "    client-fingerprint: %s\n", p.FP)
		}
	case "none", "":
	default:
		b.WriteString("    tls: true\n")
	}
	if sec == "reality" {
		b.WriteString("    reality-opts:\n")
		if p.PBK != "" {
			fmt.Fprintf(b, "      public-key: %s\n", p.PBK)
		}
		if p.SID != "" {
			fmt.Fprintf(b, "      short-id: %s\n", p.SID)
		}
	}
	if network == "ws" && p.Path != "" {
		b.WriteString("    ws-opts:\n")
		fmt.Fprintf(b, "      path: %s\n", p.Path)
		if p.Host != "" {
			b.WriteString("      headers:\n")
			fmt.Fprintf(b, "        Host: %s\n", p.Host)
		}
	}
	if network == "grpc" {
		svc := p.GrpcServiceName
		if svc == "" {
			svc = p.Path
		}
		if svc != "" {
			b.WriteString("    grpc-opts:\n")
			fmt.Fprintf(b, "      grpc-service-name: %s\n", svc)
		}
	}
	if p.AllowInsecure {
		b.WriteString("    skip-cert-verify: true\n")
	}
	if enc := strings.ToLower(p.PacketEncoding); enc != "" && enc != "none" {
		fmt.Fprintf(b, "    packet-encoding: %s\n", enc)
	}
}

func appendRuDirectRules(b *strings.Builder) {
	// GEOIP only: bundled GeoSite.dat from mihomo often lacks the "ru" list (fatal on start).
	b.WriteString("\nrules:\n")
	b.WriteString("  - GEOIP,ru,DIRECT\n")
	b.WriteString("  - GEOIP,private,DIRECT\n")
	b.WriteString("  - MATCH,PROXY\n")
}
