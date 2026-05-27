package pseudogui

import (
	"strings"
	"testing"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/ui/model"
)

func TestRegression_SyncSimpleToast_liveError(t *testing.T) {
	v := &simpleView{
		conn: model.ConnectionUI{ErrorText: "timeout"},
		set:  model.SettingsUI{},
	}
	syncSimpleToast(v)
	if v.toast != "timeout" || !v.toastErr {
		t.Fatalf("toast=%q err=%v", v.toast, v.toastErr)
	}
}

func TestRegression_SyncSimpleToast_clearsPinnedWhenSettingOff(t *testing.T) {
	v := &simpleView{
		conn:     model.ConnectionUI{},
		set:      model.SettingsUI{UIKeepErrorsOnScreen: false},
		toast:    "old",
		toastErr: true,
	}
	syncSimpleToast(v)
	if v.toast != "" || v.toastErr {
		t.Fatalf("toast=%q err=%v", v.toast, v.toastErr)
	}
}

func TestRegression_SyncSimpleToast_keepsPinnedWhenSettingOn(t *testing.T) {
	v := &simpleView{
		conn:     model.ConnectionUI{},
		set:      model.SettingsUI{UIKeepErrorsOnScreen: true},
		toast:    "pinned",
		toastErr: true,
	}
	syncSimpleToast(v)
	if v.toast != "pinned" || !v.toastErr {
		t.Fatalf("toast=%q err=%v", v.toast, v.toastErr)
	}
}

func TestRegression_SimpleToastLine_prefersLiveError(t *testing.T) {
	v := simpleView{
		conn:  model.ConnectionUI{ErrorText: "live"},
		toast: "pinned",
	}
	if got := simpleToastLine(v); got != "live" {
		t.Fatalf("got %q", got)
	}
}

func TestRegression_PickList_pickerOptionLabel(t *testing.T) {
	if got := pickerOptionLabel(2, "[current] foo"); got != "[current] foo" {
		t.Fatalf("bracket: %q", got)
	}
	if got := pickerOptionLabel(3, "RU direct"); got != "3  RU direct" {
		t.Fatalf("numbered: %q", got)
	}
}

func TestRegression_UIConnecting_busyWithoutConnected(t *testing.T) {
	c := model.ConnectionUI{Busy: true}
	if !uiConnecting(c) {
		t.Fatal("busy")
	}
	c = model.ConnectionUI{Connected: true, Busy: true}
	if uiConnecting(c) {
		t.Fatal("connected wins")
	}
}

func TestRegression_UIConnecting_stateConnecting(t *testing.T) {
	c := model.ConnectionUI{State: apiclient.StateConnecting}
	if !uiConnecting(c) {
		t.Fatal("connecting state")
	}
}

func TestRegression_StatusTitle_connecting(t *testing.T) {
	c := model.ConnectionUI{State: apiclient.StateConnecting, Busy: true}
	if got := statusTitle(c); got != "Подключение…" {
		t.Fatalf("got %q", got)
	}
}

func TestRegression_ParseChoice_menuActions(t *testing.T) {
	got, ok := ParseChoice("p")
	if !ok || got != ActionProfiles {
		t.Fatalf("profiles: %q ok=%v", got, ok)
	}
	got, ok = ParseChoice("q")
	if !ok || got != ActionQuit {
		t.Fatalf("quit: %q ok=%v", got, ok)
	}
}

func TestRegression_ProbeBlock_skipsOnError(t *testing.T) {
	c := model.ConnectionUI{
		Connected:   true,
		ErrorText:   "fail",
		ProfileName: "p1",
		ProxyName:   "proxy",
		ProbeText:   "probe",
	}
	if probe := probeBlock(c); probe != "" {
		t.Fatalf("expected empty, got %q", probe)
	}
}

func TestRegression_ProbeBlock_proxyLabelWhenConnected(t *testing.T) {
	c := model.ConnectionUI{
		Connected:   true,
		ProfileName: "My profile",
		ProxyName:   "tag_p1",
	}
	probe := probeBlock(c)
	if !strings.Contains(probe, "Прокси:") {
		t.Fatalf("probe=%q", probe)
	}
}
