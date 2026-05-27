package presenter

import (
	"context"
	"errors"
	"testing"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/ui/model"
)

func TestRegression_DaemonBlip_suppressedWhileConnecting(t *testing.T) {
	p := &Presenter{
		conn: model.ConnectionUI{
			State: apiclient.StateConnecting,
			Busy:  true,
		},
		connecting: true,
	}
	p.setDaemonUnreachable(errors.New("daemon not running: connection refused"))
	if p.conn.ErrorText != "" {
		t.Fatalf("expected suppressed error, got %q", p.conn.ErrorText)
	}
}

func TestRegression_DaemonBlip_suppressedWhileConnected(t *testing.T) {
	p := &Presenter{
		conn: model.ConnectionUI{
			Connected: true,
			State:     apiclient.StateConnected,
		},
	}
	p.setDaemonUnreachable(errors.New("daemon not running: connection refused"))
	if p.conn.ErrorText != "" {
		t.Fatalf("expected suppressed error while connected, got %q", p.conn.ErrorText)
	}
	if !p.conn.Connected {
		t.Fatal("connected flag must survive SSE blip")
	}
}

func TestRegression_DaemonBlip_showsWhenIdleUnreachable(t *testing.T) {
	p := &Presenter{conn: model.ConnectionUI{State: apiclient.StateIdle}}
	p.setDaemonUnreachable(errors.New("daemon not running: connection refused"))
	if p.conn.ErrorText == "" {
		t.Fatal("expected error text when idle and daemon unreachable")
	}
	if p.conn.Connected {
		t.Fatal("should clear connected on unreachable when idle")
	}
}

func TestRegression_DaemonBlip_timeoutNotTreatedAsDown(t *testing.T) {
	p := &Presenter{conn: model.ConnectionUI{State: apiclient.StateIdle}}
	p.setDaemonUnreachable(errors.New("context deadline exceeded"))
	if p.conn.ErrorText == "" {
		t.Fatal("timeout should surface as error text")
	}
	if !p.conn.Connected && p.conn.State != apiclient.StateIdle {
		t.Fatalf("unexpected state change on timeout: %#v", p.conn)
	}
}

func TestRegression_ShuttingDown_suppressesErrors(t *testing.T) {
	p := &Presenter{conn: model.ConnectionUI{State: apiclient.StateIdle}}
	p.SetShuttingDown(true)
	p.setDaemonUnreachable(errors.New("daemon not running"))
	if p.conn.ErrorText != "" {
		t.Fatalf("shutdown should suppress errors, got %q", p.conn.ErrorText)
	}
}

func TestRegression_RefreshIfIdle_singleFlight(t *testing.T) {
	p := &Presenter{}
	p.mu.Lock()
	p.refreshBusy = true
	p.mu.Unlock()
	p.refreshIfIdle(context.Background())
	p.mu.Lock()
	stillBusy := p.refreshBusy
	p.mu.Unlock()
	if !stillBusy {
		t.Fatal("refreshIfIdle must not clear refreshBusy when already busy")
	}
}
