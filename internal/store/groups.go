package store

import "context"

type Group struct {
	ID               int64
	Name             string
	SubscriptionLink string
}

func (s *Store) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, subscription_link FROM groups ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.SubscriptionLink); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) CreateGroup(ctx context.Context, name, link string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO groups (name, subscription_link) VALUES (?, ?)`, name, link)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
