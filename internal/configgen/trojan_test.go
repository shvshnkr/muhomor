package configgen

import "testing"

func TestParseTrojanURI(t *testing.T) {
	uri := "trojan://pass@1.2.3.4:8443?sni=example.com&alpn=h2,http/1.1&fp=qq#node1"
	p, err := ParseTrojanURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	if p.Password != "pass" || p.Server != "1.2.3.4" || p.Port != 8443 {
		t.Fatalf("%+v", p)
	}
}
