package subscription

import (
	"context"
	"strings"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/store"
)

// RepairTruncatedURIs fixes profiles stored as "://host…" after old ParseLines stripped the scheme.
func RepairTruncatedURIs(ctx context.Context, st *store.Store) (int, error) {
	if st == nil {
		return 0, nil
	}
	profiles, err := st.ListAllProfiles(ctx)
	if err != nil {
		return 0, err
	}
	fixed := 0
	for _, p := range profiles {
		if !strings.HasPrefix(p.URI, "://") {
			continue
		}
		typ, uri, ok := inferSchemeForTruncatedURI(p.URI)
		if !ok {
			continue
		}
		if err := st.UpdateProfileTypeURI(ctx, p.ID, typ, uri); err != nil {
			return fixed, err
		}
		fixed++
	}
	return fixed, nil
}

func inferSchemeForTruncatedURI(truncated string) (typ, uri string, ok bool) {
	// vless first — hysteria parsers are permissive and may false-match vless hosts.
	for _, sch := range []string{"vless", "trojan", "hysteria2", "hysteria"} {
		candidate := sch + truncated
		if !canParseScheme(sch, candidate) {
			continue
		}
		return sch, candidate, true
	}
	return "", truncated, false
}

func canParseScheme(sch, uri string) bool {
	switch sch {
	case "vless":
		_, err := configgen.ParseVLESSURI(uri)
		return err == nil
	case "trojan":
		_, err := configgen.ParseTrojanURI(uri)
		return err == nil
	case "hysteria", "hysteria2":
		_, err := configgen.ParseHysteriaURI(uri)
		return err == nil
	default:
		return false
	}
}
