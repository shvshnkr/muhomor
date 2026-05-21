package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/muhomor/muhomor/internal/reachability"
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
	DNSFakeIP              bool
	ChainProfileIDs        []int64
	MultipathEnabled       bool
	MultipathPreset        string // low | normal | high
	MultipathWLEmergencyOnly bool
	WLBuiltinConnectEnabled  bool
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
	out.MultipathEnabled = s.MultipathEnabled(ctx)
	out.MultipathPreset = s.MultipathPreset(ctx)
	out.MultipathWLEmergencyOnly = s.MultipathWLEmergencyOnly(ctx)
	out.WLBuiltinConnectEnabled = s.WLBuiltinConnectEnabled(ctx)
	return out, nil
}

// ConnectionTestURL returns Dahusim DataStore.connectionTestURL (default cp.cloudflare.com).
func (s *Store) ConnectionTestURL(ctx context.Context) string {
	if v, _ := s.GetKV(ctx, KeyConnectionTestURL); v != "" {
		return v
	}
	return reachability.ConnectionTestURL
}

// ConnectionTestTimeoutMs returns Dahusim DataStore.connectionTestTimeout (default 3000).
func (s *Store) ConnectionTestTimeoutMs(ctx context.Context) int {
	return intKV(ctx, s, KeyConnectionTestTimeoutMs, 3000)
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
	_ = s.SetKV(ctx, KeyMultipathEnabled, boolStr(set.MultipathEnabled))
	preset := set.MultipathPreset
	if preset != "low" && preset != "high" {
		preset = "normal"
	}
	_ = s.SetKV(ctx, KeyMultipathPreset, preset)
	_ = s.SetKV(ctx, KeyMultipathWLEmergency, boolStr(set.MultipathWLEmergencyOnly))
	_ = s.SetKV(ctx, KeyWLBuiltinConnectEnabled, boolStr(set.WLBuiltinConnectEnabled))
	return nil
}

// EnsureInboundCredentials returns stored inbound auth (no auto-generation on desktop).
// Android builds may set credentials explicitly; localhost mixed-port stays open when both empty.
func (s *Store) EnsureInboundCredentials(ctx context.Context) (user, pass string, err error) {
	set, err := s.LoadSettings(ctx)
	if err != nil {
		return "", "", err
	}
	return set.InboundUser, set.InboundPassword, nil
}

// ClearAutoInboundCredentials clears inbound auth on desktop (localhost mixed-port without login).
func (s *Store) ClearAutoInboundCredentials(ctx context.Context) error {
	set, err := s.LoadSettings(ctx)
	if err != nil {
		return err
	}
	if set.InboundUser == "" && set.InboundPassword == "" {
		return nil
	}
	set.InboundUser = ""
	set.InboundPassword = ""
	return s.SaveSettings(ctx, set)
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
