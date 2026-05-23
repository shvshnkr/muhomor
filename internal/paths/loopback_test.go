package paths

import "testing"

func TestMihomoControllerHost_default(t *testing.T) {
	t.Setenv("MUHOMOR_MIHOMO_CONTROLLER_HOST", "")
	t.Setenv("MUHOMOR_LOOPBACK_HOST", "")
	if got := MihomoControllerHost(); got != DefaultMihomoControllerHost {
		t.Fatalf("got %q", got)
	}
	if DefaultExternalController() != "127.0.0.6:9090" {
		t.Fatal(DefaultExternalController())
	}
}

func TestMihomoControllerHost_override(t *testing.T) {
	t.Setenv("MUHOMOR_MIHOMO_CONTROLLER_HOST", "127.0.0.7")
	if MihomoControllerHost() != "127.0.0.7" {
		t.Fatal("override")
	}
}

func TestMixedBindHost_separateFromController(t *testing.T) {
	t.Setenv("MUHOMOR_MIXED_BIND_HOST", "")
	if MixedBindHost() != "127.0.0.1" {
		t.Fatalf("mixed bind %q", MixedBindHost())
	}
	if MixedBindHost() == MihomoControllerHost() {
		t.Fatal("mixed port and controller host must differ by default")
	}
}
