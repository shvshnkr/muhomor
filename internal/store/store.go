package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Store SQLite persistence.
type Store struct {
	db *sql.DB
}

type Profile struct {
	ID              int64
	Name            string
	Type            string
	URI             string
	Enabled         bool
	LastDelayMs     int
	LastError       string
	Status          int
	Ping            int
	UserOrder       int64
	GroupID         int64
	WhitelistMarked bool
	WLBuiltinPool   bool
	RuExitMarked    bool
}

func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		s, err := openStore(dbPath)
		if err == nil {
			return s, nil
		}
		lastErr = err
		if attempt == 0 && sqliteRecoverable(err) {
			removeSQLiteSidecars(dbPath)
			continue
		}
		return nil, err
	}
	return nil, lastErr
}

func openStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.quickCheck(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) quickCheck() error {
	var status string
	if err := s.db.QueryRow(`PRAGMA quick_check`).Scan(&status); err != nil {
		return err
	}
	if status != "ok" {
		return fmt.Errorf("database disk image is malformed (%s)", status)
	}
	return nil
}

func sqliteRecoverable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "database disk image is malformed") ||
		strings.Contains(msg, "file is not a database") ||
		strings.Contains(msg, "malformed database schema")
}

func removeSQLiteSidecars(dbPath string) {
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
}

func (s *Store) Close() error {
	_, _ = s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	return s.db.Close()
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(schemaSQL); err != nil {
		return err
	}
	return migrate(s.db)
}

func (s *Store) UpsertProfile(ctx context.Context, name, typ, uri string) (int64, error) {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO profiles (name, type, uri) VALUES (?, ?, ?)
		 ON CONFLICT(uri) DO UPDATE SET name = excluded.name`, name, typ, uri)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM profiles WHERE uri = ?`, uri).Scan(&id)
	return id, err
}

func (s *Store) ListEnabledProfiles(ctx context.Context) ([]Profile, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, type, uri, enabled, last_delay_ms, last_error
		 FROM profiles WHERE enabled = 1 ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Profile
	for rows.Next() {
		var p Profile
		var en int
		if err := rows.Scan(&p.ID, &p.Name, &p.Type, &p.URI, &en, &p.LastDelayMs, &p.LastError); err != nil {
			return nil, err
		}
		p.Enabled = en == 1
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) UpdateProfileDelay(ctx context.Context, id int64, delayMs int, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE profiles SET last_delay_ms = ?, last_error = ? WHERE id = ?`,
		delayMs, errMsg, id)
	return err
}

func (s *Store) SetKV(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO kv (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (s *Store) GetKV(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM kv WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func (s *Store) CurrentProfileID(ctx context.Context) (int64, error) {
	v, err := s.GetKV(ctx, "current_profile_id")
	if err != nil || v == "" {
		return 0, err
	}
	var id int64
	_, scanErr := fmt.Sscan(v, &id)
	return id, scanErr
}

func (s *Store) SetCurrentProfileID(ctx context.Context, id int64) error {
	return s.SetKV(ctx, "current_profile_id", fmt.Sprintf("%d", id))
}

func (s *Store) ProfileByID(ctx context.Context, id int64) (Profile, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, type, uri, enabled, last_delay_ms, last_error,
		        COALESCE(status,1), COALESCE(ping,0), COALESCE(user_order,0),
		        COALESCE(group_id,0), COALESCE(whitelist_marked,0), COALESCE(wl_builtin_pool,0),
		        COALESCE(ru_exit_marked,0)
		 FROM profiles WHERE id = ?`, id)
	if err != nil {
		return Profile{}, err
	}
	defer rows.Close()
	list, err := scanProfiles(rows)
	if err != nil || len(list) == 0 {
		return Profile{}, err
	}
	return list[0], nil
}
