package bootstrap

import (
	"context"
	"fmt"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/store"
)

// Mirrors WhitelistBuiltinProxies + WhitelistBuiltinVlessShareLines (Dahusim).
const WLGroupName = store.BuiltinWLGroupName

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
		id, err := st.UpsertProfileInGroup(ctx, gid, def.Name, "trojan", uri, int64(i+1), def.WLOnlyPool, def.WLOnlyPool)
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
		_, _ = st.UpsertProfileInGroup(ctx, gid, name, "vless", line, 100+int64(i+1), true, true)
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
