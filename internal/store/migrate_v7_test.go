package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestMigrate_v7_channelColumns(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "v7.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 6`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	var ver int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != schemaVersion {
		t.Fatalf("version %d want %d", ver, schemaVersion)
	}
	var ruCol int
	_ = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('profiles') WHERE name = 'ru_exit_marked'`).Scan(&ruCol)
	if ruCol != 1 {
		t.Fatal("ru_exit_marked column missing")
	}
	st, err := Open(filepath.Join(dir, "open.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	id, err := st.UpsertProfile(ctx, "ch1", "vless", "vless://u@1.2.3.4:443?security=none#t")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateChannelMetrics(ctx, id, ChannelMetrics{
		EWMAGoodputKbps: 12000, EWMALossPermille: 10, State: ChannelHealthy, LastSampleAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	m, err := st.ChannelMetricsByID(ctx, id)
	if err != nil || m.EWMAGoodputKbps != 12000 {
		t.Fatalf("%+v err=%v", m, err)
	}
}
