package presenter

import (
	"strings"
	"testing"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/ui/model"
)

func TestRegression_FormatProbeProgress_tcpOnlyWhenSchedulerOff(t *testing.T) {
	pr := &apiclient.ProbeProgress{TotalEnabled: 10, Alive: 0, SchedulerEnabled: false}
	got := formatProbeProgress(pr, apiclient.ServiceStatus{})
	if got != "Подбор по TCP (URL-тест недоступен)" {
		t.Fatalf("got %q", got)
	}
}

func TestRegression_FormatProbeProgress_degradedSuffix(t *testing.T) {
	pr := &apiclient.ProbeProgress{TotalEnabled: 5, Alive: 3, SchedulerEnabled: true}
	st := apiclient.ServiceStatus{ConnectedDegraded: true}
	got := formatProbeProgress(pr, st)
	if got == "" || !strings.Contains(got, "degraded") {
		t.Fatalf("got %q", got)
	}
}

func TestRegression_FormatProbeProgress_noLiveServers(t *testing.T) {
	got := formatProbeProgress(nil, apiclient.ServiceStatus{
		VerificationPhase: apiclient.VerificationNoLiveServers,
	})
	if got != "Живых серверов не найдено" {
		t.Fatalf("got %q", got)
	}
}

func TestRegression_Emit_shuttingDownStripsErrorAndActivity(t *testing.T) {
	var gotConn model.ConnectionUI
	p := &Presenter{
		shuttingDown: true,
		conn: model.ConnectionUI{
			ErrorText:    "daemon not running",
			ActivityText: "Подключение…",
			Busy:         true,
			LastPingError: "timeout",
		},
		OnUI: func(c model.ConnectionUI, _ model.SettingsUI) { gotConn = c },
	}
	p.emit()
	if gotConn.ErrorText != "" || gotConn.ActivityText != "" || gotConn.Busy || gotConn.LastPingError != "" {
		t.Fatalf("emit during shutdown: %#v", gotConn)
	}
}

func TestRegression_ApplyStatus_setsProbeText(t *testing.T) {
	p := &Presenter{}
	st := apiclient.ServiceStatus{
		State:     apiclient.StateConnected,
		Connected: true,
		Probe:     &apiclient.ProbeProgress{TotalEnabled: 4, Alive: 2, SchedulerEnabled: true},
	}
	p.applyStatus(st)
	if p.conn.ProbeText == "" {
		t.Fatal("expected probe text")
	}
}

func TestRegression_IsStaleStartupActivity(t *testing.T) {
	if !isStaleStartupActivity("Подключение к серверу…") {
		t.Fatal("connect activity is stale after connect")
	}
	if isStaleStartupActivity("TCP 12/64") {
		t.Fatal("probe progress is not stale startup")
	}
}
