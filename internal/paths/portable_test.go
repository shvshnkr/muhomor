package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPortableKitMarkersPresent(t *testing.T) {
	dir := t.TempDir()
	if portableKitMarkersPresent(dir) {
		t.Fatal("empty dir should not be kit")
	}
	_ = os.MkdirAll(filepath.Join(dir, "bin"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "bin", "mihomo.exe"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "muhomor.exe"), []byte("x"), 0o644)
	if !portableKitMarkersPresent(dir) {
		t.Fatal("expected windows kit layout detected")
	}
	_ = os.Remove(filepath.Join(dir, "muhomor.exe"))
	_ = os.WriteFile(filepath.Join(dir, "bin", "mihomo"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "muhomor"), []byte("x"), 0o644)
	if !portableKitMarkersPresent(dir) {
		t.Fatal("expected linux kit layout detected")
	}
}

func TestResolveDataDir_priority(t *testing.T) {
	t.Setenv("MUHOMOR_DATA_DIR", "")
	if got := ResolveDataDir("/a/b"); got != "/a/b" {
		t.Fatalf("explicit: %q", got)
	}
	dir := t.TempDir()
	t.Setenv("MUHOMOR_DATA_DIR", filepath.Join(dir, "fromenv"))
	if got := ResolveDataDir(""); got != filepath.Join(dir, "fromenv") {
		t.Fatalf("env: %q", got)
	}
}

func TestDefaultMixedPort(t *testing.T) {
	if DefaultMixedPort(9999) != 9999 {
		t.Fatal("requested port wins")
	}
}
