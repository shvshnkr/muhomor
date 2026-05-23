package subscription

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_loadSubscriptionLinksFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subscriptions.txt")
	if err := os.WriteFile(path, []byte("# comment\n\nhttps://example.com/sub\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	links, err := loadSubscriptionLinksFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0] != "https://example.com/sub" {
		t.Fatalf("links=%v", links)
	}
}
