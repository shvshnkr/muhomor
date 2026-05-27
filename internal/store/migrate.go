package store

import (
	"database/sql"
	"fmt"
)

const schemaVersion = 8

func migrate(db *sql.DB) error {
	var v int
	_ = db.QueryRow(`PRAGMA user_version`).Scan(&v)
	if v >= schemaVersion {
		return nil
	}
	stmts := []string{
		`ALTER TABLE profiles ADD COLUMN status INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE profiles ADD COLUMN ping INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE profiles ADD COLUMN user_order INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE profiles ADD COLUMN group_id INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE profiles ADD COLUMN whitelist_marked INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE groups ADD COLUMN last_updated_at TEXT NOT NULL DEFAULT ''`,
	}
	for _, s := range stmts {
		_, _ = db.Exec(s) // ignore duplicate column on re-run
	}
	if v < 3 {
		_, _ = db.Exec(`ALTER TABLE profiles ADD COLUMN wl_builtin_pool INTEGER NOT NULL DEFAULT 0`)
	}
	if v < 4 {
		_, _ = db.Exec(`ALTER TABLE groups ADD COLUMN kind TEXT NOT NULL DEFAULT 'manual'`)
		_, _ = db.Exec(`UPDATE groups SET kind = 'subscription' WHERE subscription_link != '' AND name != ?`, BuiltinWLGroupName)
	}
	if v < 5 {
		probeCols := []string{
			`ALTER TABLE profiles ADD COLUMN probe_state INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE profiles ADD COLUMN probe_last_checked_at TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE profiles ADD COLUMN probe_last_ok_at TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE profiles ADD COLUMN probe_last_fail_at TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE profiles ADD COLUMN probe_fail_streak INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE profiles ADD COLUMN probe_success_window INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE profiles ADD COLUMN probe_ewma_delay_ms INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE profiles ADD COLUMN probe_last_error_class TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE profiles ADD COLUMN probe_next_probe_at TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE profiles ADD COLUMN probe_source_priority INTEGER NOT NULL DEFAULT 0`,
		}
		for _, s := range probeCols {
			_, _ = db.Exec(s)
		}
		_, _ = db.Exec(`UPDATE profiles SET probe_state = ?, probe_ewma_delay_ms = ping
			WHERE status = ? AND ping > 0`, ProbeAlive, StatusAvailable)
		_, _ = db.Exec(`UPDATE profiles SET probe_state = ? WHERE status = ?`, ProbeDead, StatusUnreachable)
		_, _ = db.Exec(`UPDATE profiles SET probe_source_priority = ? WHERE wl_builtin_pool = 1`, ProbeSourceBuiltin)
	}
	if v < 6 {
		for _, old := range LegacySwordwareSubscriptionURLs {
			_, _ = db.Exec(`UPDATE groups SET subscription_link = ? WHERE subscription_link = ?`,
				SwordwareSubscriptionURL, old)
		}
	}
	if v < 8 {
		_, _ = db.Exec(`ALTER TABLE profiles ADD COLUMN ru_exit_marked INTEGER NOT NULL DEFAULT 0`)
	}
	if v < 7 {
		chCols := []string{
			`ALTER TABLE profiles ADD COLUMN ch_ewma_goodput_kbps INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE profiles ADD COLUMN ch_ewma_loss_permille INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE profiles ADD COLUMN ch_ewma_jitter_ms INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE profiles ADD COLUMN ch_ewma_queue_delay_ms INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE profiles ADD COLUMN ch_channel_state INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE profiles ADD COLUMN ch_last_sample_at TEXT NOT NULL DEFAULT ''`,
		}
		for _, s := range chCols {
			_, _ = db.Exec(s)
		}
		// Seed goodput from probe EWMA delay where available.
		_, _ = db.Exec(`UPDATE profiles SET ch_ewma_goodput_kbps =
			CASE WHEN probe_ewma_delay_ms > 0 THEN MAX(100, 8000 / probe_ewma_delay_ms) ELSE 0 END
			WHERE ch_ewma_goodput_kbps = 0 AND probe_ewma_delay_ms > 0`)
	}
	_, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion))
	return err
}
