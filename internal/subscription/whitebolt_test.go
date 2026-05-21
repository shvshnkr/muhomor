package subscription

import (
	"context"
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestIsWhiteBoltWLGroup(t *testing.T) {
	if !IsWhiteBoltWLGroup(WhiteBoltWLGroupPrefix + "test") {
		t.Fatal("expected white bolt group")
	}
	if IsWhiteBoltWLGroup("Quick Subscription 1") {
		t.Fatal("quick sub is not white bolt")
	}
}

func TestHasWhiteBoltWLGroups_empty(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ok, err := HasWhiteBoltWLGroups(context.Background(), st)
	if err != nil || ok {
		t.Fatalf("has=%v err=%v", ok, err)
	}
}
