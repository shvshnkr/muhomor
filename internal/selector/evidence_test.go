package selector

import "testing"

func TestSetAndGetProbeEvidence(t *testing.T) {
	s := &Selector{}
	ev := ProbeEvidence{TCPOK: 10, URLOK: 0, PoolSize: 32, Degraded: true}
	s.setProbeEvidence(ev)
	got := s.LastProbeEvidence()
	if got.TCPOK != ev.TCPOK || got.URLOK != ev.URLOK || got.PoolSize != ev.PoolSize || got.Degraded != ev.Degraded {
		t.Fatalf("unexpected probe evidence: %#v", got)
	}
}
