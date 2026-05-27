package wailsapp

import "testing"

func TestRegression_StartHiddenEnabled_flag(t *testing.T) {
	t.Setenv("MUHOMOR_GUI_START_HIDDEN", "")
	if !StartHiddenEnabled(true) {
		t.Fatal("flag true")
	}
	if StartHiddenEnabled(false) {
		t.Fatal("flag false without env")
	}
}

func TestRegression_StartHiddenEnabled_env(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "yes", "on"} {
		t.Setenv("MUHOMOR_GUI_START_HIDDEN", v)
		if !StartHiddenEnabled(false) {
			t.Fatalf("env %q", v)
		}
	}
	t.Setenv("MUHOMOR_GUI_START_HIDDEN", "0")
	if StartHiddenEnabled(false) {
		t.Fatal("env 0")
	}
}

func TestRegression_DesktopRunOptions_startHiddenWiring(t *testing.T) {
	opt := DesktopRunOptions{StartHidden: true}
	if !opt.StartHidden {
		t.Fatal("field not set")
	}
}
