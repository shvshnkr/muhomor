package controller

import "testing"

func TestConnectionVerifierTransitions(t *testing.T) {
	v := NewConnectionVerifier()
	v.StartAttempt()
	v.ObserveSelection(12, 0, 48)
	v.MarkTransportAlive("mihomo_api_ready")

	s := v.Snapshot()
	if s.Phase != VerificationTransportAlive {
		t.Fatalf("phase = %s, want %s", s.Phase, VerificationTransportAlive)
	}
	if !s.ConnectedDegraded {
		t.Fatalf("expected degraded transport phase")
	}
	if s.LiveServersConfirmed {
		t.Fatalf("unexpected live confirmation in degraded phase")
	}

	v.MarkQualityVerified("delay_probe_ok")
	s = v.Snapshot()
	if !s.ConnectedVerified || !s.LiveServersConfirmed {
		t.Fatalf("expected verified+confirmed after quality success: %#v", s)
	}
}

func TestConnectionVerifierNoLiveServers(t *testing.T) {
	v := NewConnectionVerifier()
	v.MarkNoLiveServers("all probes dead")
	s := v.Snapshot()
	if s.Phase != VerificationNoLiveServers {
		t.Fatalf("phase = %s, want %s", s.Phase, VerificationNoLiveServers)
	}
	if s.ConnectedVerified || s.LiveServersConfirmed {
		t.Fatalf("unexpected connected flags for no_live_servers")
	}
}
