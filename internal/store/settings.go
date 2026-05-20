package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
)

// Service mode (DataStore Key.MODE_*).
const (
	ServiceModeProxy = "proxy"
	ServiceModeVPN   = "vpn"
)

const (
	KeyServiceMode       = "service_mode"
	KeyMixedPort         = "mixed_port"
	KeySocksPort         = "socks_port"
	KeyHTTPPort          = "http_port"
	KeyAllowLAN          = "allow_lan"
	KeyInboundUser       = "inbound_username"
	KeyInboundPassword   = "inbound_password"
	KeyTunEnable         = "tun_enable"
	KeyTunStack          = "tun_stack"
	KeyTunDNSHijack      = "tun_dns_hijack"
	KeyTunMTU            = "tun_mtu"
	KeyDNSFakeIP         = "dns_fake_ip"
	KeyChainProfileIDs   = "chain_profile_ids"
	KeyLastAssetUpdateAt = "last_asset_update_at"
)

// Settings persisted preferences (DataStore subset).
type Settings struct {
	ServiceMode     string
	RouteQuickProfile int
	MixedPort       int
	SocksPort       int
	HTTPPort        int
	AllowLAN        bool
	InboundUser     string
	InboundPassword string
	TunEnable       bool
	TunStack        string
	TunDNSHijack    bool
	TunMTU          int
	DNSFakeIP       bool
	ChainProfileIDs []int64
}

func DefaultSettings() Settings {
	return Settings{
		ServiceMode:       ServiceModeProxy,
		RouteQuickProfile: RouteQuickRuDirectOnly,
		MixedPort:         7890,
		SocksPort:         0,
		HTTPPort:          0,
		TunStack:          "system",
		TunMTU:            9000,
	}
}

func (s *Store) LoadSettings(ctx context.Context) (Settings, error) {
	out := DefaultSettings()
	if v, _ := s.GetKV(ctx, KeyServiceMode); v != "" {
		out.ServiceMode = v
	}
	if v, _ := s.GetKV(ctx, KeyRouteQuickProfile); v != "" {
		out.RouteQuickProfile, _ = strconv.Atoi(v)
	}
	out.MixedPort = intKV(ctx, s, KeyMixedPort, out.MixedPort)
	out.SocksPort = intKV(ctx, s, KeySocksPort, 0)
	out.HTTPPort = intKV(ctx, s, KeyHTTPPort, 0)
	out.AllowLAN = boolKV(ctx, s, KeyAllowLAN, false)
	out.InboundUser, _ = s.GetKV(ctx, KeyInboundUser)
	out.InboundPassword, _ = s.GetKV(ctx, KeyInboundPassword)
	out.TunEnable = boolKV(ctx, s, KeyTunEnable, out.ServiceMode == ServiceModeVPN)
	out.TunStack = strKV(ctx, s, KeyTunStack, out.TunStack)
	out.TunDNSHijack = boolKV(ctx, s, KeyTunDNSHijack, true)
	out.TunMTU = intKV(ctx, s, KeyTunMTU, out.TunMTU)
	out.DNSFakeIP = boolKV(ctx, s, KeyDNSFakeIP, false)
	if ids, _ := s.GetKV(ctx, KeyChainProfileIDs); ids != "" {
		out.ChainProfileIDs = parseIDList(ids)
	}
	return out, nil
}

func (s *Store) SaveSettings(ctx context.Context, set Settings) error {
	_ = s.SetKV(ctx, KeyServiceMode, set.ServiceMode)
	_ = s.SetKV(ctx, KeyRouteQuickProfile, strconv.Itoa(set.RouteQuickProfile))
	_ = s.SetKV(ctx, KeyMixedPort, strconv.Itoa(set.MixedPort))
	_ = s.SetKV(ctx, KeySocksPort, strconv.Itoa(set.SocksPort))
	_ = s.SetKV(ctx, KeyHTTPPort, strconv.Itoa(set.HTTPPort))
	_ = s.SetKV(ctx, KeyAllowLAN, boolStr(set.AllowLAN))
	_ = s.SetKV(ctx, KeyInboundUser, set.InboundUser)
	_ = s.SetKV(ctx, KeyInboundPassword, set.InboundPassword)
	_ = s.SetKV(ctx, KeyTunEnable, boolStr(set.TunEnable))
	_ = s.SetKV(ctx, KeyTunStack, set.TunStack)
	_ = s.SetKV(ctx, KeyTunDNSHijack, boolStr(set.TunDNSHijack))
	_ = s.SetKV(ctx, KeyTunMTU, strconv.Itoa(set.TunMTU))
	_ = s.SetKV(ctx, KeyDNSFakeIP, boolStr(set.DNSFakeIP))
	if len(set.ChainProfileIDs) > 0 {
		_ = s.SetKV(ctx, KeyChainProfileIDs, formatIDList(set.ChainProfileIDs))
	}
	return nil
}

// EnsureInboundCredentials generates random user/pass if both empty (RU-OPTIMIZATION / zapret advisory).
func (s *Store) EnsureInboundCredentials(ctx context.Context) (user, pass string, err error) {
	set, err := s.LoadSettings(ctx)
	if err != nil {
		return "", "", err
	}
	if set.InboundUser != "" || set.InboundPassword != "" {
		return set.InboundUser, set.InboundPassword, nil
	}
	set.InboundUser = "muhomor"
	set.InboundPassword = randomToken(16)
	if err := s.SaveSettings(ctx, set); err != nil {
		return "", "", err
	}
	return set.InboundUser, set.InboundPassword, nil
}

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func intKV(ctx context.Context, s *Store, key string, def int) int {
	v, _ := s.GetKV(ctx, key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func boolKV(ctx context.Context, s *Store, key string, def bool) bool {
	v, _ := s.GetKV(ctx, key)
	if v == "" {
		return def
	}
	return v == "true" || v == "1"
}

func strKV(ctx context.Context, s *Store, key string, def string) string {
	v, _ := s.GetKV(ctx, key)
	if v == "" {
		return def
	}
	return v
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func parseIDList(s string) []int64 {
	var out []int64
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if id, err := strconv.ParseInt(p, 10, 64); err == nil {
			out = append(out, id)
		}
	}
	return out
}

func formatIDList(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(parts, ",")
}
