package configgen

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// TrojanProfile parsed trojan outbound.
type TrojanProfile struct {
	Name     string
	Password string
	Server   string
	Port     int
	SNI      string
	ALPN     string
	FP       string
}

// ParseTrojanURI parses trojan://password@host:port?params#name
func ParseTrojanURI(raw string) (TrojanProfile, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "trojan://") {
		return TrojanProfile{}, fmt.Errorf("not a trojan URI")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return TrojanProfile{}, err
	}
	pass := ""
	if u.User != nil {
		pass, _ = u.User.Password()
		if pass == "" {
			pass = u.User.Username()
		}
	}
	host := u.Hostname()
	portStr := u.Port()
	if host == "" || pass == "" {
		return TrojanProfile{}, fmt.Errorf("missing host or password")
	}
	port := 443
	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return TrojanProfile{}, err
		}
	}
	q := u.Query()
	p := TrojanProfile{
		Name:     u.Fragment,
		Password: pass,
		Server:   host,
		Port:     port,
		SNI:      firstNonEmpty(q.Get("sni"), q.Get("peer"), host),
		ALPN:     q.Get("alpn"),
		FP:       q.Get("fp"),
	}
	if p.Name == "" {
		p.Name = net.JoinHostPort(host, portStr)
	}
	return p, nil
}

func writeTrojanProxy(b *strings.Builder, name string, p TrojanProfile) {
	yamlNameLine(b, name)
	b.WriteString("    type: trojan\n")
	b.WriteString("    udp: true\n")
	fmt.Fprintf(b, "    server: %s\n", p.Server)
	fmt.Fprintf(b, "    port: %d\n", p.Port)
	fmt.Fprintf(b, "    password: %s\n", p.Password)
	b.WriteString("    tls: true\n")
	if p.SNI != "" {
		fmt.Fprintf(b, "    sni: %s\n", p.SNI)
	}
	if p.ALPN != "" {
		b.WriteString("    alpn:\n")
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
}

// FormatTrojanURI builds internal storage URI for trojan profiles.
func FormatTrojanURI(p TrojanProfile) string {
	q := url.Values{}
	if p.SNI != "" {
		q.Set("sni", p.SNI)
	}
	if p.ALPN != "" {
		q.Set("alpn", p.ALPN)
	}
	if p.FP != "" {
		q.Set("fp", p.FP)
	}
	u := &url.URL{
		Scheme:   "trojan",
		User:     url.UserPassword(p.Password, ""),
		Host:     fmt.Sprintf("%s:%d", p.Server, p.Port),
		RawQuery: q.Encode(),
		Fragment: p.Name,
	}
	return u.String()
}
