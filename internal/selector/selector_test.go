package selector

import (
	"testing"
	"time"

	"github.com/muhomor/muhomor/internal/profileclass"
	"github.com/muhomor/muhomor/internal/store"
)

func TestCompositeScore_prefersURL(t *testing.T) {
	p := store.Profile{ID: 1}
	tcp := map[int64]int{1: 50}
	url := map[int64]int{1: 120}
	if compositeScore(p, tcp, url, nil, false, false, nil) != 120 {
		t.Fatalf("expected url latency to win")
	}
}

func TestURLTestCandidates_tcpLiveFirst(t *testing.T) {
	pool := []store.Profile{
		{ID: 1, Name: "dead"},
		{ID: 2, Name: "fast"},
		{ID: 3, Name: "slow"},
	}
	tcp := map[int64]int{2: 10, 3: 200}
	got := urlTestCandidates(pool, tcp, nil, 2)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("order=%v,%v want 2,3", got[0].ID, got[1].ID)
	}
}

func TestCompositeScore_ruExitOnlyBLPenalty(t *testing.T) {
	p := store.Profile{ID: 5, Name: "Russia [01]", RuExitMarked: true}
	tcp := map[int64]int{5: 50}
	blOK := map[int64]bool{5: false}
	if compositeScore(p, tcp, nil, blOK, true, false, nil) != 1<<30 {
		t.Fatalf("ru_exit without BL should be dead score")
	}
	blOK[5] = true
	if compositeScore(p, tcp, nil, blOK, true, false, nil) == 1<<30 {
		t.Fatal("ru_exit with BL ok should use TCP synthetic")
	}
	p2 := store.Profile{ID: 6, Name: "Austria [01]"}
	tcp2 := map[int64]int{6: 50}
	blOK2 := map[int64]bool{6: false}
	if compositeScore(p2, tcp2, nil, blOK2, true, false, nil) == 1<<30 {
		t.Fatal("non-uplink should not be penalized in uplink scope")
	}
	_ = profileclass.IsRuExitMarked(p)
}

func TestCompositeScore_allScopeBLPenalty(t *testing.T) {
	p := store.Profile{ID: 6, Name: "Austria [01]"}
	tcp := map[int64]int{6: 50}
	blOK := map[int64]bool{6: false}
	if compositeScore(p, tcp, nil, blOK, false, false, nil) != 1<<30 {
		t.Fatal("all scope should penalize without BL pass")
	}
}

func TestCompositeScore_blTagUplinkPenalty(t *testing.T) {
	bl := store.Profile{ID: 7, Name: "Anycast [BL]"}
	tcp := map[int64]int{7: 10}
	blOK := map[int64]bool{7: false}
	if compositeScore(bl, tcp, nil, blOK, true, false, nil) != 1<<30 {
		t.Fatal("[bl] uplink without BL pass should be dead score")
	}
	blOK[7] = true
	if compositeScore(bl, tcp, nil, blOK, true, false, nil) == 1<<30 {
		t.Fatal("[bl] with BL ok should rank")
	}
}

func TestCompositeScore_tcpSynthetic(t *testing.T) {
	p := store.Profile{ID: 2}
	tcp := map[int64]int{2: 100}
	s := compositeScore(p, tcp, nil, nil, false, false, nil)
	if s < 1000 || s > 2000 {
		t.Fatalf("unexpected synthetic score %d", s)
	}
}

func TestCompositeScore_tcpOnlyDegraded_demotesUnknown(t *testing.T) {
	fast := store.Profile{ID: 1}
	slow := store.Profile{ID: 2}
	tcp := map[int64]int{1: 50, 2: 100}
	urlAlive := map[int64]struct{}{1: {}}
	sFast := compositeScore(fast, tcp, nil, nil, false, true, urlAlive)
	sSlow := compositeScore(slow, tcp, nil, nil, false, true, urlAlive)
	if sFast >= sSlow {
		t.Fatalf("url_alive profile should rank before TCP-only: fast=%d slow=%d", sFast, sSlow)
	}
}

func TestClassifyPrepareDecision(t *testing.T) {
	if got := classifyPrepareDecision(0, 0); got != PrepareDecisionHardDead {
		t.Fatalf("want hard_dead, got %s", got)
	}
	if got := classifyPrepareDecision(4, 0); got != PrepareDecisionDegraded {
		t.Fatalf("want degraded, got %s", got)
	}
	if got := classifyPrepareDecision(4, 2); got != PrepareDecisionOK {
		t.Fatalf("want ok, got %s", got)
	}
}

func TestTelegramCircuitOpensOnLowRatio(t *testing.T) {
	s := &Selector{}
	s.recordTelegramBatch(30, 1)
	if !s.telegramCircuitOpen() {
		t.Fatal("expected telegram circuit to open")
	}
}

func TestTelegramCircuitExpires(t *testing.T) {
	s := &Selector{}
	s.targetCircuitOpenUntil = time.Now().Add(-time.Second)
	if s.telegramCircuitOpen() {
		t.Fatal("expected expired circuit to close")
	}
}
