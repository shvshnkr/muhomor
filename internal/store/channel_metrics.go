package store

import (
	"context"
	"database/sql"
	"time"
)

// Channel tier states (multipath aggregation).
const (
	ChannelHealthy    = 0
	ChannelCongested  = 1
	ChannelDegraded   = 2
	ChannelEmergency  = 3
)

// ChannelMetrics persisted per profile (EWMA windows).
type ChannelMetrics struct {
	EWMAGoodputKbps   int
	EWMALossPermille  int // 0..1000
	EWMAJitterMs      int
	EWMAQueueDelayMs  int
	State             int
	LastSampleAt      time.Time
}

func ChannelStateName(s int) string {
	switch s {
	case ChannelCongested:
		return "congested"
	case ChannelDegraded:
		return "degraded"
	case ChannelEmergency:
		return "emergency"
	default:
		return "healthy"
	}
}

func (s *Store) ChannelMetricsByID(ctx context.Context, id int64) (ChannelMetrics, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(ch_ewma_goodput_kbps,0), COALESCE(ch_ewma_loss_permille,0),
		       COALESCE(ch_ewma_jitter_ms,0), COALESCE(ch_ewma_queue_delay_ms,0),
		       COALESCE(ch_channel_state,0), COALESCE(ch_last_sample_at,'')
		FROM profiles WHERE id = ?`, id)
	return scanChannelMetricsRow(row)
}

func scanChannelMetricsRow(row *sql.Row) (ChannelMetrics, error) {
	var m ChannelMetrics
	var at string
	if err := row.Scan(&m.EWMAGoodputKbps, &m.EWMALossPermille, &m.EWMAJitterMs, &m.EWMAQueueDelayMs, &m.State, &at); err != nil {
		return m, err
	}
	m.LastSampleAt = timeFromDB(at)
	return m, nil
}

func (s *Store) UpdateChannelMetrics(ctx context.Context, id int64, m ChannelMetrics) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE profiles SET
		  ch_ewma_goodput_kbps = ?,
		  ch_ewma_loss_permille = ?,
		  ch_ewma_jitter_ms = ?,
		  ch_ewma_queue_delay_ms = ?,
		  ch_channel_state = ?,
		  ch_last_sample_at = ?
		WHERE id = ?`,
		m.EWMAGoodputKbps, m.EWMALossPermille, m.EWMAJitterMs, m.EWMAQueueDelayMs, m.State,
		timeToDB(m.LastSampleAt), id)
	return err
}
