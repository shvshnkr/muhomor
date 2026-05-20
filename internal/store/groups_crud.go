package store

import (
	"context"
	"fmt"
)

// BuiltinWLGroupName is the built-in whitelist pool group (must match bootstrap).
const BuiltinWLGroupName = "Built-in (simple mode helpers)"

// DeleteGroup removes a group if not built-in WL pool.
func (s *Store) DeleteGroup(ctx context.Context, id int64) error {
	g, err := s.groupByID(ctx, id)
	if err != nil {
		return err
	}
	if g.Name == BuiltinWLGroupName {
		return fmt.Errorf("built-in group cannot be deleted")
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM profiles WHERE group_id = ?`, id)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM groups WHERE id = ?`, id)
	return err
}

func (s *Store) groupByID(ctx context.Context, id int64) (Group, error) {
	var g Group
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, subscription_link, COALESCE(kind,'manual') FROM groups WHERE id = ?`, id).
		Scan(&g.ID, &g.Name, &g.SubscriptionLink, &g.Kind)
	return g, err
}

// SetGroupSubscription updates subscription URL for a group.
func (s *Store) SetGroupSubscription(ctx context.Context, id int64, link string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE groups SET subscription_link = ? WHERE id = ?`, link, id)
	return err
}

// SetGroupKind updates group kind (subscription / manual).
func (s *Store) SetGroupKind(ctx context.Context, id int64, kind string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE groups SET kind = ? WHERE id = ?`, kind, id)
	return err
}

// TouchGroupUpdated sets last_updated_at for a group.
func (s *Store) TouchGroupUpdated(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE groups SET last_updated_at = datetime('now') WHERE id = ?`, id)
	return err
}

// DeleteProfile removes one profile (not WL builtin).
func (s *Store) DeleteProfile(ctx context.Context, id int64) error {
	var wl int
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(wl_builtin_pool,0) FROM profiles WHERE id = ?`, id).Scan(&wl)
	if err != nil {
		return err
	}
	if wl == 1 {
		return fmt.Errorf("built-in profile cannot be deleted")
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM profiles WHERE id = ?`, id)
	return err
}

// ProfileCountByGroup returns enabled profile count per group id.
func (s *Store) ProfileCountByGroup(ctx context.Context) (map[int64]int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT group_id, COUNT(*) FROM profiles GROUP BY group_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var gid int64
		var n int
		if err := rows.Scan(&gid, &n); err != nil {
			return nil, err
		}
		out[gid] = n
	}
	return out, rows.Err()
}

// SortGroupProfilesByDelay sets user_order by ascending last_delay_ms (0 at end).
func (s *Store) SortGroupProfilesByDelay(ctx context.Context, groupID int64) error {
	list, err := s.ListProfilesByGroup(ctx, groupID)
	if err != nil {
		return err
	}
	sortProfilesByDelay(list)
	for i, p := range list {
		_, err := s.db.ExecContext(ctx,
			`UPDATE profiles SET user_order = ? WHERE id = ?`, int64(i+1), p.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func sortProfilesByDelay(list []Profile) {
	// simple stable sort: ok delay first ascending, failures last
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			si, sj := scoreDelay(list[i]), scoreDelay(list[j])
			if sj < si {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
}

func scoreDelay(p Profile) int {
	if p.LastDelayMs <= 0 {
		return 1 << 30
	}
	return p.LastDelayMs
}
