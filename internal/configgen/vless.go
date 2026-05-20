package configgen

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// VLESSProfile parsed from vless:// URI (subset for MVP).
type VLESSProfile struct {
	Name   string
	UUID   string
	Server string
	Port   int
	Flow   string
	Network string
	Security string
	SNI    string
	FP     string
	PBK    string
	SID    string
	ALPN   string
	Path   string
	Host   string
}

// ParseVLESSURI parses vless://uuid@host:port?query#name
func ParseVLESSURI(raw string) (VLESSProfile, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "vless://") {
		return VLESSProfile{}, fmt.Errorf("not a vless URI")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return VLESSProfile{}, err
	}
	if u.User == nil || u.User.Username() == "" {
		return VLESSProfile{}, fmt.Errorf("missing uuid")
	}
	host := u.Hostname()
	portStr := u.Port()
	if host == "" {
		return VLESSProfile{}, fmt.Errorf("missing host")
	}
	port := 443
	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return VLESSProfile{}, fmt.Errorf("invalid port: %w", err)
		}
	}
	q := u.Query()
	p := VLESSProfile{
		Name:     strings.TrimSpace(u.Fragment),
		UUID:     u.User.Username(),
		Server:   host,
		Port:     port,
		Flow:     q.Get("flow"),
		Network:  firstNonEmpty(q.Get("type"), "tcp"),
		Security: firstNonEmpty(q.Get("security"), "none"),
		SNI:      firstNonEmpty(q.Get("sni"), q.Get("peer"), host),
		FP:       q.Get("fp"),
		PBK:      q.Get("pbk"),
		SID:      q.Get("sid"),
		ALPN:     q.Get("alpn"),
		Path:     q.Get("path"),
		Host:     firstNonEmpty(q.Get("host"), q.Get("authority")),
	}
	if p.Name == "" {
		p.Name = net.JoinHostPort(host, portStr)
	}
	return p, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
