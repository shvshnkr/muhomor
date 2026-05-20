//go:build integration

package subscription

import (
	"context"
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestFetchMifaLearnUA(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	gid, err := st.CreateGroup(ctx, "mifa", "https://mifa.world/vless", store.GroupKindSubscription)
	if err != nil {
		t.Fatal(err)
	}
	u := &Updater{Store: st}
	lines, ua, err := u.fetchWithLearnedUA(ctx, gid, "https://mifa.world/vless")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 1 {
		t.Fatal("no lines")
	}
	if ua == "" {
		t.Fatal("expected learned ua")
	}
	t.Logf("lines=%d ua=%s mode=%s", len(lines), ua, UAModeLabel(ua))
}
