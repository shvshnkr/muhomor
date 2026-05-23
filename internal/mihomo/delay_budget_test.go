package mihomo

import "testing"

func TestGroupDelayAPITimeout_scalesWithCount(t *testing.T) {
	small := GroupDelayAPITimeout(3000, 1)
	large := GroupDelayAPITimeout(3000, 70)
	if large <= small {
		t.Fatalf("large=%d small=%d", large, small)
	}
	if large < 30000 {
		t.Fatalf("expected >=30s for 70 proxies, got %d", large)
	}
}
