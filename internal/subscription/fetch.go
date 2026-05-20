package subscription

import (
	"encoding/base64"
	"strings"
)

// NormalizeSubscriptionBody decodes base64 subscription payloads (mifa.world, many airports).
func NormalizeSubscriptionBody(raw []byte) []byte {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return raw
	}
	if strings.Contains(s, "://") {
		return raw
	}
	if dec, ok := tryBase64Decode(s); ok && strings.Contains(string(dec), "://") {
		return dec
	}
	return raw
}

func tryBase64Decode(s string) ([]byte, bool) {
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, s)
	if s == "" {
		return nil, false
	}
	for _, dec := range []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		base64.URLEncoding.DecodeString,
		base64.RawURLEncoding.DecodeString,
	} {
		if b, err := dec(s); err == nil && len(b) > 0 {
			return b, true
		}
	}
	return nil, false
}
