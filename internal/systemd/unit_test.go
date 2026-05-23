package systemd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestUnitDir_scopes(t *testing.T) {
	home, err := UnitDir("user")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(home, filepath.Join(".config", "systemd", "user")) {
		t.Fatalf("user dir: %s", home)
	}
	sys, err := UnitDir("system")
	if err != nil {
		t.Fatal(err)
	}
	if sys != "/etc/systemd/system" {
		t.Fatalf("system dir: %s", sys)
	}
}

func TestRenderUnit_appendsDaemon(t *testing.T) {
	body := RenderUnit("/usr/bin/muhomor -dir /data")
	if !strings.Contains(body, "ExecStart=/usr/bin/muhomor -dir /data --daemon") {
		t.Fatalf("missing --daemon: %s", body)
	}
	body2 := RenderUnit("/usr/bin/muhomor --daemon -dir /data")
	if strings.Count(body2, "--daemon") != 1 {
		t.Fatalf("duplicate daemon flag: %s", body2)
	}
}
