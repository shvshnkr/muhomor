package store

import "database/sql"

const schemaVersion = 3

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
	_, err := db.Exec(`PRAGMA user_version = 3`)
	return err
}
