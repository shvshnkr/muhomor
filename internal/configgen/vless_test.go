package configgen

import (
	"strings"
	"testing"
)

func TestParseVLESSURI(t *testing.T) {
	uri := "vless://11111111-2222-3333-4444-555555555555@example.com:443?security=tls&sni=example.com&type=ws&path=/vless#test-node"
	p, err := ParseVLESSURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	if p.UUID != "11111111-2222-3333-4444-555555555555" {
		t.Fatalf("uuid: %s", p.UUID)
	}
	if p.Server != "example.com" || p.Port != 443 {
		t.Fatalf("host/port: %s:%d", p.Server, p.Port)
	}
	if p.Name != "test-node" {
		t.Fatalf("name: %s", p.Name)
	}
}

func TestBuildFromVLESS_RuDirectOrder(t *testing.T) {
	p := VLESSProfile{Name: "n1", UUID: "u", Server: "h", Port: 443, Security: "tls"}
	yaml, _, err := BuildFromVLESS(p, DefaultBuildOptions())
	if err != nil {
		t.Fatal(err)
	}
	ruIdx := strings.Index(yaml, "GEOIP,ru,DIRECT")
	matchIdx := strings.Index(yaml, "MATCH,PROXY")
	if ruIdx < 0 || matchIdx < 0 || ruIdx > matchIdx {
		t.Fatalf("rule order wrong:\n%s", yaml)
	}
}
