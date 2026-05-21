package store

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

const (
	KeyMultipathEnabled     = "multipath_enabled"
	KeyMultipathPreset      = "multipath_preset"
	KeyMultipathWLEmergency = "multipath_wl_emergency_only"
	KeyMultipathChannelPool = "multipath_channel_pool"
	KeyMultipathLastReason  = "multipath_last_reason"
	KeyMultipathStats       = "multipath_stats"
)

// MultipathStats JSON snapshot for API.
type MultipathStats struct {
	Enabled         bool   `json:"enabled"`
	Preset          string `json:"preset,omitempty"`
	WLEmergencyOnly bool   `json:"wl_emergency_only"`
	ActiveChannels  int    `json:"active_channels"`
	HealthyChannels int    `json:"healthy_channels"`
	LastReason      string `json:"last_reason,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
}

func (s *Store) MultipathEnabled(ctx context.Context) bool {
	return boolKV(ctx, s, KeyMultipathEnabled, false)
}

func (s *Store) MultipathWLEmergencyOnly(ctx context.Context) bool {
	return boolKV(ctx, s, KeyMultipathWLEmergency, true)
}

func (s *Store) MultipathPreset(ctx context.Context) string {
	v, _ := s.GetKV(ctx, KeyMultipathPreset)
	switch v {
	case "low", "high":
		return v
	default:
		return "normal"
	}
}

func (s *Store) SetMultipathStats(ctx context.Context, st MultipathStats) error {
	st.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return s.SetKV(ctx, KeyMultipathStats, string(b))
}

func (s *Store) GetMultipathStats(ctx context.Context) MultipathStats {
	raw, _ := s.GetKV(ctx, KeyMultipathStats)
	if raw == "" {
		return MultipathStats{
			Enabled:         s.MultipathEnabled(ctx),
			Preset:          s.MultipathPreset(ctx),
			WLEmergencyOnly: s.MultipathWLEmergencyOnly(ctx),
		}
	}
	var st MultipathStats
	if json.Unmarshal([]byte(raw), &st) != nil {
		return MultipathStats{Enabled: s.MultipathEnabled(ctx)}
	}
	return st
}

func (s *Store) SetMultipathChannelPool(ctx context.Context, ids []int64) error {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return s.SetKV(ctx, KeyMultipathChannelPool, strings.Join(parts, ","))
}

func (s *Store) SetMultipathLastReason(ctx context.Context, reason string) error {
	return s.SetKV(ctx, KeyMultipathLastReason, reason)
}
