package subscription

import (
	"strings"
	"testing"
)

func TestParseLines_keepsScheme(t *testing.T) {
	body := "vless://uuid@1.2.3.4:443?security=none#n1\n" +
		"junk vless://uuid@5.6.7.8:8443?security=none#n2\n"
	lines, err := ParseLines(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("lines=%d", len(lines))
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "vless://") {
			t.Fatalf("stripped scheme: %q", l)
		}
	}
}
