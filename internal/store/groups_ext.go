package store

import "context"

func (s *Store) EnsureGroup(ctx context.Context, name string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM groups WHERE name = ?`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO groups (name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpsertProfileInGroup(ctx context.Context, groupID int64, name, typ, uri string, order int64, wlBuiltin, whitelistMarked bool) (int64, error) {
	wlB, wlM := 0, 0
	if wlBuiltin {
		wlB = 1
	}
	if whitelistMarked {
		wlM = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO profiles (name, type, uri, group_id, user_order, wl_builtin_pool, whitelist_marked)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(uri) DO UPDATE SET name=excluded.name, group_id=excluded.group_id,
		   user_order=excluded.user_order, wl_builtin_pool=excluded.wl_builtin_pool,
		   whitelist_marked=excluded.whitelist_marked`,
		name, typ, uri, groupID, order, wlB, wlM)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM profiles WHERE uri = ?`, uri).Scan(&id)
	return id, err
}

func (s *Store) ListProfilesByGroup(ctx context.Context, groupID int64) ([]Profile, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, type, uri, enabled, last_delay_ms, last_error,
		        COALESCE(status,1), COALESCE(ping,0), COALESCE(user_order,0),
		        COALESCE(group_id,0), COALESCE(whitelist_marked,0), COALESCE(wl_builtin_pool,0)
		 FROM profiles WHERE group_id = ? ORDER BY user_order`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProfilesExt(rows)
}

func scanProfilesExt(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]Profile, error) {
	var out []Profile
	for rows.Next() {
		var p Profile
		var en, wlMark, wlPool int
		if err := rows.Scan(&p.ID, &p.Name, &p.Type, &p.URI, &en, &p.LastDelayMs, &p.LastError,
			&p.Status, &p.Ping, &p.UserOrder, &p.GroupID, &wlMark, &wlPool); err != nil {
			return nil, err
		}
		p.Enabled = en == 1
		p.WhitelistMarked = wlMark == 1
		p.WLBuiltinPool = wlPool == 1
		out = append(out, p)
	}
	return out, rows.Err()
}

// PruneGroupProfilesNotInURIs removes subscription rows absent from the latest fetch (keeps wl_builtin_pool).
func (s *Store) PruneGroupProfilesNotInURIs(ctx context.Context, groupID int64, keepURIs map[string]struct{}) (int, error) {
	if len(keepURIs) == 0 {
		return 0, nil
	}
	list, err := s.ListProfilesByGroup(ctx, groupID)
	if err != nil {
		return 0, err
	}
	var n int
	for _, p := range list {
		if p.WLBuiltinPool {
			continue
		}
		if _, ok := keepURIs[p.URI]; ok {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM profiles WHERE id = ? AND group_id = ?`, p.ID, groupID); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
