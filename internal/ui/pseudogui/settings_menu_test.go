package pseudogui

import (
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func Test_bulkLBStrategyNames(t *testing.T) {
	if store.NormalizeBulkLBStrategy(store.BulkLBStickySessions) != store.BulkLBStickySessions {
		t.Fatal("sticky")
	}
	if store.NormalizeBulkLBStrategy("consistent") != store.BulkLBConsistentHash {
		t.Fatal("consistent alias")
	}
}
