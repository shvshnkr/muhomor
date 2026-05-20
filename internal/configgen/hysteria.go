package configgen

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// HysteriaProfile from hysteria2:// or hysteria:// URI (Phase 3 subset).
type HysteriaProfile struct {
	Name     string
	Server   string
	Port     int
	Password string
	SNI      string
	Insecure bool
}

// ParseHysteriaURI parses hysteria(2) subscription lines.
func ParseHysteriaURI(raw string) (HysteriaProfile, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "hysteria://") && !strings.HasPrefix(raw, "hysteria2://") {
		return HysteriaProfile{}, fmt.Errorf("not hysteria uri")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return HysteriaProfile{}, err
	}
	pass, _ := u.User.Password()
	if pass == "" && u.User != nil {
		pass = u.User.Username()
	}
	port := 443
	if p := u.Port(); p != "" {
		port, _ = strconv.Atoi(p)
	}
	q := u.Query()
	return HysteriaProfile{
		Name:     u.Fragment,
		Server:   u.Hostname(),
		Port:     port,
		Password: pass,
		SNI:      firstNonEmpty(q.Get("sni"), u.Hostname()),
		Insecure: q.Get("insecure") == "1" || q.Get("allowInsecure") == "1",
	}, nil
}

func writeHysteria2Proxy(b *strings.Builder, name string, p HysteriaProfile) {
	fmt.Fprintf(b, "  - name: %s\n", name)
	b.WriteString("    type: hysteria2\n")
	fmt.Fprintf(b, "    server: %s\n", p.Server)
	fmt.Fprintf(b, "    port: %d\n", p.Port)
	if p.Password != "" {
		fmt.Fprintf(b, "    password: %s\n", p.Password)
	}
	if p.SNI != "" {
		fmt.Fprintf(b, "    sni: %s\n", p.SNI)
	}
	if p.Insecure {
		b.WriteString("    skip-cert-verify: true\n")
	}
}
