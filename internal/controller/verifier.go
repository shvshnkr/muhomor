package controller

import (
	"fmt"
	"sync"
	"time"
)

type VerificationPhase string

const (
	VerificationUnknown        VerificationPhase = "unknown"
	VerificationTransportAlive VerificationPhase = "transport_alive"
	VerificationQualityOK      VerificationPhase = "quality_verified"
	VerificationNoLiveServers  VerificationPhase = "no_live_servers"
)

type VerificationEvidence struct {
	TCPOK            int
	URLOK            int
	PoolSize         int
	Degraded         bool
	LastSuccessAtUTC string
}

type VerificationSnapshot struct {
	Phase                VerificationPhase
	ConnectedVerified    bool
	ConnectedDegraded    bool
	LiveServersConfirmed bool
	Reason               string
	Evidence             VerificationEvidence
	UpdatedAtUTC         string
}

type ConnectionVerifier struct {
	mu         sync.Mutex
	phase      VerificationPhase
	reason     string
	evidence   VerificationEvidence
	updatedAt  time.Time
	lastGoodAt time.Time
}

func NewConnectionVerifier() *ConnectionVerifier {
	return &ConnectionVerifier{phase: VerificationUnknown}
}

func (v *ConnectionVerifier) StartAttempt() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.phase = VerificationUnknown
	v.reason = ""
	v.updatedAt = time.Now()
}

func (v *ConnectionVerifier) ObserveSelection(tcpOK, urlOK, pool int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.evidence.TCPOK = tcpOK
	v.evidence.URLOK = urlOK
	v.evidence.PoolSize = pool
	v.evidence.Degraded = tcpOK > 0 && urlOK == 0
	v.updatedAt = time.Now()
}

func (v *ConnectionVerifier) MarkTransportAlive(reason string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.phase = VerificationTransportAlive
	v.reason = reason
	v.updatedAt = time.Now()
}

func (v *ConnectionVerifier) MarkQualityVerified(reason string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.phase = VerificationQualityOK
	v.reason = reason
	v.lastGoodAt = time.Now()
	v.updatedAt = v.lastGoodAt
}

func (v *ConnectionVerifier) MarkNoLiveServers(reason string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.phase = VerificationNoLiveServers
	v.reason = reason
	v.updatedAt = time.Now()
}

func (v *ConnectionVerifier) Snapshot() VerificationSnapshot {
	v.mu.Lock()
	defer v.mu.Unlock()
	evidence := v.evidence
	if !v.lastGoodAt.IsZero() {
		evidence.LastSuccessAtUTC = v.lastGoodAt.UTC().Format(time.RFC3339)
	}
	phase := v.phase
	if phase == "" {
		phase = VerificationUnknown
	}
	return VerificationSnapshot{
		Phase:                phase,
		ConnectedVerified:    phase == VerificationQualityOK,
		ConnectedDegraded:    phase == VerificationTransportAlive && evidence.Degraded,
		LiveServersConfirmed: phase == VerificationQualityOK,
		Reason:               v.reason,
		Evidence:             evidence,
		UpdatedAtUTC:         formatTime(v.updatedAt),
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (s VerificationSnapshot) String() string {
	return fmt.Sprintf("phase=%s verified=%t degraded=%t", s.Phase, s.ConnectedVerified, s.ConnectedDegraded)
}
