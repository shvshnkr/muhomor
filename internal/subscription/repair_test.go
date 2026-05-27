package subscription

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestRepairTruncatedURIs(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	gid, err := st.CreateGroup(ctx, "g", "http://example/sub", store.GroupKindSubscription)
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertProfileInGroup(ctx, gid, "p1", "", "://uuid@1.2.3.4:443?security=none#t", 1, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	n, err := RepairTruncatedURIs(ctx, st)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	list, _ := st.ListProfilesByGroup(ctx, gid)
	if len(list) != 1 || list[0].Type != "vless" || !strings.HasPrefix(list[0].URI, "vless://") {
		t.Fatalf("profile=%+v", list[0])
	}
}
