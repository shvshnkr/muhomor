package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (s *Store) UpdateProfileTypeURI(ctx context.Context, id int64, typ, uri string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE profiles SET type = ?, uri = ? WHERE id = ?`,
		typ, uri, id)
	return err
}

func (s *Store) ListAllProfiles(ctx context.Context) ([]Profile, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, type, uri, enabled, last_delay_ms, last_error,
		        COALESCE(status,1), COALESCE(ping,0), COALESCE(user_order,0),
		        COALESCE(group_id,0), COALESCE(whitelist_marked,0), COALESCE(wl_builtin_pool,0)
		 FROM profiles ORDER BY user_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProfiles(rows)
}

func (s *Store) UpdateProfileProbe(ctx context.Context, id int64, delayMs int, status int, errMsg string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE profiles SET last_delay_ms = ?, ping = ?, status = ?, last_error = ? WHERE id = ?`,
		delayMs, delayMs, status, errMsg, id)
	if err != nil {
		return err
	}
	prev, err := s.ProbeMetaByID(ctx, id)
	if err != nil {
		prev = LegacyStatusToProbe(status, delayMs)
	}
	if delayMs > 0 {
		prev = applyProbeOnURLSuccess(prev, delayMs, now)
	} else {
		prev = bumpProbeOnURLFail(prev, now)
	}
	return s.UpdateProfileProbeMeta(ctx, id, prev)
}

func (s *Store) SetSelectedProxy(ctx context.Context, id int64) error {
	return s.SetKV(ctx, KeySelectedProxy, strconv.FormatInt(id, 10))
}

func (s *Store) SelectedProxy(ctx context.Context) (int64, error) {
	v, err := s.GetKV(ctx, KeySelectedProxy)
	if err != nil || v == "" {
		return 0, err
	}
	return strconv.ParseInt(v, 10, 64)
}

func (s *Store) RouteQuickProfile(ctx context.Context) (int, error) {
	v, err := s.GetKV(ctx, KeyRouteQuickProfile)
	if err != nil || v == "" {
		return RouteQuickRuDirectOnly, nil
	}
	n, _ := strconv.Atoi(v)
	return n, nil
}

func (s *Store) SetRouteQuickProfile(ctx context.Context, profile int) error {
	return s.SetKV(ctx, KeyRouteQuickProfile, strconv.Itoa(profile))
}

func (s *Store) SetFallbackQueue(ctx context.Context, ids []int64) error {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	_ = s.SetKV(ctx, KeyAutoSelectFallbackIndex, "0")
	return s.SetKV(ctx, KeyAutoSelectFallbackQueue, strings.Join(parts, ","))
}

func (s *Store) FallbackQueue(ctx context.Context) ([]int64, error) {
	v, _ := s.GetKV(ctx, KeyAutoSelectFallbackQueue)
	if v == "" {
		return nil, nil
	}
	var out []int64
	for _, p := range strings.Split(v, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err == nil {
			out = append(out, id)
		}
	}
	return out, nil
}

// FallbackSkip returns true to skip a queue entry (dead, cooldown, etc.).
type FallbackSkip func(id int64) bool

func (s *Store) TryMoveFallback(ctx context.Context, currentID int64) (int64, bool) {
	return s.TryMoveFallbackSkip(ctx, currentID, nil)
}

func (s *Store) TryMoveFallbackSkip(ctx context.Context, currentID int64, skip FallbackSkip) (int64, bool) {
	q, err := s.FallbackQueue(ctx)
	if err != nil || len(q) == 0 {
		return 0, false
	}
	idx := 0
	v, _ := s.GetKV(ctx, KeyAutoSelectFallbackIndex)
	if v != "" {
		idx, _ = strconv.Atoi(v)
	}
	start := idx
	found := false
	for i, id := range q {
		if id == currentID {
			start = i + 1
			found = true
			break
		}
	}
	if !found {
		start = idx
	}
	if start < idx {
		start = idx
	}
	for i := start; i < len(q); i++ {
		next := q[i]
		if skip != nil && skip(next) {
			continue
		}
		_ = s.SetKV(ctx, KeyAutoSelectFallbackIndex, strconv.Itoa(i))
		_ = s.SetSelectedProxy(ctx, next)
		return next, true
	}
	return 0, false
}

func (s *Store) SetProfileEnabled(ctx context.Context, id int64, enabled bool) error {
	en := 0
	if enabled {
		en = 1
	}
	res, err := s.db.ExecContext(ctx, `UPDATE profiles SET enabled = ? WHERE id = ?`, en, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("profile %d not found", id)
	}
	return nil
}

func (s *Store) SetLastKnownGood(ctx context.Context, id int64) error {
	return s.SetKV(ctx, KeyAutoSelectLastKnownGood, strconv.FormatInt(id, 10))
}

func (s *Store) LastKnownGood(ctx context.Context) (int64, error) {
	v, _ := s.GetKV(ctx, KeyAutoSelectLastKnownGood)
	if v == "" {
		return 0, nil
	}
	return strconv.ParseInt(v, 10, 64)
}

func (p Profile) IsSubscriptionWhitelistMarked() bool {
	n := strings.ToLower(p.Name)
	return strings.Contains(n, "white lists") ||
		strings.Contains(n, "white list") ||
		strings.Contains(n, "whitelist") ||
		p.WhitelistMarked
}

func scanProfiles(rows *sql.Rows) ([]Profile, error) {
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
