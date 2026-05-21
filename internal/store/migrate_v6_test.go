package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrate_v6_swordwareSubscription(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "pre-v6.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	old := LegacySwordwareSubscriptionURLs[0]
	if _, err := db.Exec(`INSERT INTO groups (name, subscription_link) VALUES (?, ?)`,
		"Quick Subscription 3", old); err != nil {
		t.Fatal(err)
	}
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	var link string
	if err := db.QueryRow(`SELECT subscription_link FROM groups WHERE name = ?`, "Quick Subscription 3").Scan(&link); err != nil {
		t.Fatal(err)
	}
	if link != SwordwareSubscriptionURL {
		t.Fatalf("link %q want %q", link, SwordwareSubscriptionURL)
	}
	var ver int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != schemaVersion {
		t.Fatalf("version %d want %d", ver, schemaVersion)
	}
}

func TestMigrate_v6_fromV5_onlyUpdatesLink(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "v5.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	old := LegacySwordwareSubscriptionURLs[0]
	if _, err := db.Exec(`INSERT INTO groups (name, subscription_link) VALUES ('Swordware', ?)`, old); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 5`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	var link string
	if err := db.QueryRow(`SELECT subscription_link FROM groups LIMIT 1`).Scan(&link); err != nil {
		t.Fatal(err)
	}
	if link != SwordwareSubscriptionURL {
		t.Fatalf("link %q", link)
	}
}
