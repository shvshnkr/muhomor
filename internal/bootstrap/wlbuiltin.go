package bootstrap

import (
	"context"
	"fmt"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/store"
)

// Mirrors WhitelistBuiltinProxies + WhitelistBuiltinVlessShareLines (Dahusim).
const WLGroupName = "Built-in (simple mode helpers)"

const wlSharedPassword = "Qfw0MqoyNkSvqjRhZ_x5WNM3V_tF6q"

type builtinTrojan struct {
	Name       string
	Address    string
	Port       int
	SNI        string
	ALPN       string
	WLOnlyPool bool
}

var builtinTrojans = []builtinTrojan{
	{"Simple helper PL #41", "87.239.104.97", 7443, "plthree.rushtaxi.ru", "h2,http/1.1", true},
	{"Simple helper PL #42", "109.120.190.146", 8443, "pl.serverstats.ru", "h2,http/1.1", true},
	{"Simple helper PL #43", "109.120.191.129", 8443, "pltwo.rushtaxi.ru", "h2,http/1.1", true},
	{"Simple helper PL #44", "79.174.95.188", 7443, "pltwo.rushtaxi.ru", "h2,http/1.1", true},
	{"Simple helper RU federal", "ru.federal-usa.com", 8443, "ru.federal-usa.com", "h3,h2,http/1.1", false},
}

// WL VLESS lines from WhitelistBuiltinVlessShareLines.kt (subset kept in sync).
var wlVlessLines = []string{
	"vless://c233bb45-1f51-42f6-800a-2085a22c3e6b@62.152.56.8:6443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=yandex.ru&fp=qq&pbk=JCnvoBX8E2brxjn8OB1XUnTJ0jCvLgLbkyErIIjZYnA&sid=a1b2c3d4e5f6a7b8&packetEncoding=xudp",
	"vless://85b5cb2e-2617-4930-b6a4-4aeaf3b7b9aa@89.23.100.17:443?encryption=none&flow=xtls-rprx-vision&security=tls&sni=sub.sbrf-cdn342.ru&alpn=http/1.1&fp=qq&packetEncoding=xudp",
	"vless://9f770440-7892-4bd4-9a0d-9fa30a5c5376@193.233.217.143:443?encryption=none&security=reality&sni=yahoo.com&fp=chrome&pbk=MBlHbIz4hj-uQhDA55cgoEvOlXMlXyJ9YyjDKbwt1yU&sid=5e30&packetEncoding=xudp",
	"vless://45e55198-a5ad-4f19-bb39-236822141d25@188.72.103.3:443?encryption=none&security=tls&sni=cdn.tracker.yandex.net&fp=chrome&type=ws&host=cdn.lovecrafty.link&path=/stream/updates/b66b78d7/019dfd7f-0777-6283-7287-911777c3720f4&packetEncoding=xudp",
}

// EnsureWLBuiltin syncs built-in WL pool into store (WhitelistBuiltinBootstrap).
func EnsureWLBuiltin(ctx context.Context, st *store.Store) (groupID int64, err error) {
	gid, err := st.EnsureGroup(ctx, WLGroupName)
	if err != nil {
		return 0, err
	}
	for i, def := range builtinTrojans {
		p := configgen.TrojanProfile{
			Name:     def.Name,
			Password: wlSharedPassword,
			Server:   def.Address,
			Port:     def.Port,
			SNI:      def.SNI,
			ALPN:     def.ALPN,
			FP:       "qq",
		}
		uri := configgen.FormatTrojanURI(p)
		id, err := st.UpsertProfileInGroup(ctx, gid, def.Name, "trojan", uri, int64(i+1), def.WLOnlyPool)
		if err != nil {
			return gid, err
		}
		_ = id
	}
	for i, line := range wlVlessLines {
		v, err := configgen.ParseVLESSURI(line)
		if err != nil {
			continue
		}
		name := fmt.Sprintf("WL vless #%02d", i+1)
		v.Name = name
		_, _ = st.UpsertProfileInGroup(ctx, gid, name, "vless", line, 100+int64(i+1), true)
	}
	return gid, nil
}

// WLPoolProfiles returns profiles for whitelist-only selection (4 trojans + WL vless).
func WLPoolProfiles(ctx context.Context, st *store.Store) ([]store.Profile, error) {
	groups, err := st.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	var gid int64
	for _, g := range groups {
		if g.Name == WLGroupName {
			gid = g.ID
			break
		}
	}
	if gid == 0 {
		return nil, nil
	}
	all, err := st.ListProfilesByGroup(ctx, gid)
	if err != nil {
		return nil, err
	}
	var out []store.Profile
	for _, p := range all {
		if p.WLBuiltinPool || stringsHasPrefix(p.Name, "WL vless #") || isWLTrojanName(p.Name) {
			out = append(out, p)
		}
	}
	return out, nil
}

func isWLTrojanName(name string) bool {
	for _, d := range builtinTrojans {
		if d.Name == name && d.WLOnlyPool {
			return true
		}
	}
	return false
}

func stringsHasPrefix(s, pre string) bool {
	return len(s) >= len(pre) && s[:len(pre)] == pre
}
