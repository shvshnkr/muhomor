package store

import (
	"context"
	"database/sql"
	"time"
)

func timeToDB(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func timeFromDB(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (s *Store) UpdateProfileProbeMeta(ctx context.Context, id int64, m ProbeMeta) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE profiles SET
		  probe_state = ?,
		  probe_last_checked_at = ?,
		  probe_last_ok_at = ?,
		  probe_last_fail_at = ?,
		  probe_fail_streak = ?,
		  probe_success_window = ?,
		  probe_ewma_delay_ms = ?,
		  probe_last_error_class = ?,
		  probe_next_probe_at = ?,
		  probe_source_priority = ?,
		  last_delay_ms = CASE WHEN ? > 0 THEN ? ELSE last_delay_ms END,
		  ping = CASE WHEN ? > 0 THEN ? ELSE ping END,
		  status = CASE
		    WHEN ? IN (?, ?) THEN ?
		    WHEN ? IN (?, ?) THEN ?
		    ELSE status
		  END
		WHERE id = ?`,
		m.State,
		timeToDB(m.LastCheckedAt), timeToDB(m.LastOKAt), timeToDB(m.LastFailAt),
		m.FailStreak, m.SuccessWindow, m.EWMADelayMs, m.LastErrorClass, timeToDB(m.NextProbeAt), m.SourcePriority,
		m.EWMADelayMs, m.EWMADelayMs, m.EWMADelayMs, m.EWMADelayMs,
		m.State, ProbeAlive, ProbeCandidate, StatusAvailable,
		m.State, ProbeDead, ProbeCemetery, StatusUnreachable,
		id,
	)
	return err
}

func (s *Store) ProbeMetaByID(ctx context.Context, id int64) (ProbeMeta, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(probe_state,0), COALESCE(probe_last_checked_at,''),
		       COALESCE(probe_last_ok_at,''), COALESCE(probe_last_fail_at,''),
		       COALESCE(probe_fail_streak,0), COALESCE(probe_success_window,0),
		       COALESCE(probe_ewma_delay_ms,0), COALESCE(probe_last_error_class,''),
		       COALESCE(probe_next_probe_at,''), COALESCE(probe_source_priority,0)
		FROM profiles WHERE id = ?`, id)
	return scanProbeMetaRow(row)
}

func scanProbeMetaRow(row *sql.Row) (ProbeMeta, error) {
	var m ProbeMeta
	var chk, okAt, failAt, nextAt string
	if err := row.Scan(&m.State, &chk, &okAt, &failAt, &m.FailStreak, &m.SuccessWindow,
		&m.EWMADelayMs, &m.LastErrorClass, &nextAt, &m.SourcePriority); err != nil {
		return m, err
	}
	m.LastCheckedAt = timeFromDB(chk)
	m.LastOKAt = timeFromDB(okAt)
	m.LastFailAt = timeFromDB(failAt)
	m.NextProbeAt = timeFromDB(nextAt)
	return m, nil
}

func (s *Store) CountProbeStates(ctx context.Context) (map[int]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT COALESCE(probe_state,0), COUNT(*) FROM profiles GROUP BY probe_state`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]int{}
	for rows.Next() {
		var st, n int
		if err := rows.Scan(&st, &n); err != nil {
			return nil, err
		}
		out[st] = n
	}
	return out, rows.Err()
}

func (s *Store) ListProfilesDueProbe(ctx context.Context, limit int, now time.Time) ([]Profile, error) {
	nowS := timeToDB(now)
	wlSQL := ""
	if !s.WLBuiltinConnectEnabled(ctx) {
		wlSQL = ` AND COALESCE(wl_builtin_pool,0) = 0`
	}
	q := `
		SELECT id, name, type, uri, enabled, last_delay_ms, last_error,
		       COALESCE(status,1), COALESCE(ping,0), COALESCE(user_order,0),
		       COALESCE(group_id,0), COALESCE(whitelist_marked,0), COALESCE(wl_builtin_pool,0)
		FROM profiles
		WHERE enabled = 1 AND (
		  (probe_next_probe_at = '' AND COALESCE(probe_state,0) = ?) OR
		  (probe_next_probe_at != '' AND probe_next_probe_at <= ?)
		)` + wlSQL + `
		ORDER BY probe_last_checked_at ASC, probe_state DESC, user_order
		LIMIT ?`
	rows, err := s.db.QueryContext(ctx, q, ProbeUnknown, nowS, nowS, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProfiles(rows)
}

func (s *Store) ListWarmAliveProfiles(ctx context.Context, maxAge time.Duration, limit int) ([]Profile, error) {
	now := time.Now()
	okCutoff := timeToDB(now.Add(-maxAge))
	// Reject rows whose last background check is older than 2× maxAge (stale alive).
	checkCutoff := timeToDB(now.Add(-maxAge * 2))
	wlSQL := ""
	if !s.WLBuiltinConnectEnabled(ctx) {
		wlSQL = ` AND COALESCE(wl_builtin_pool,0) = 0`
	}
	q := `
		SELECT id, name, type, uri, enabled, last_delay_ms, last_error,
		       COALESCE(status,1), COALESCE(ping,0), COALESCE(user_order,0),
		       COALESCE(group_id,0), COALESCE(whitelist_marked,0), COALESCE(wl_builtin_pool,0)
		FROM profiles
		WHERE enabled = 1 AND probe_state IN (?, ?)
		  AND probe_ewma_delay_ms > 0
		  AND probe_last_ok_at != ''
		  AND probe_last_ok_at >= ?
		  AND (probe_last_checked_at = '' OR probe_last_checked_at >= ?)` + wlSQL + `
		ORDER BY probe_ewma_delay_ms ASC, user_order
		LIMIT ?`
	rows, err := s.db.QueryContext(ctx, q, ProbeAlive, ProbeCandidate, okCutoff, checkCutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProfiles(rows)
}

func (s *Store) ProbeSchedulerEnabled(ctx context.Context) bool {
	return boolKV(ctx, s, KeyProbeSchedulerEnabled, false)
}

func (s *Store) ProbeWarmSelectEnabled(ctx context.Context) bool {
	return boolKV(ctx, s, KeyProbeWarmSelectEnabled, false)
}

func (s *Store) ProbePreset(ctx context.Context) string {
	v, _ := s.GetKV(ctx, KeyProbePreset)
	switch v {
	case "low", "high":
		return v
	default:
		return "normal"
	}
}

func (s *Store) SetProbePreset(ctx context.Context, preset string) error {
	switch preset {
	case "low", "normal", "high":
	default:
		preset = "normal"
	}
	return s.SetKV(ctx, KeyProbePreset, preset)
}

func (s *Store) BuiltinFallbackMaxPct(ctx context.Context) int {
	pct := intKV(ctx, s, KeyProbeBuiltinFallbackMaxPct, 0)
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}
