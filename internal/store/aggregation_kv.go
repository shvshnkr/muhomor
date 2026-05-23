package store

import (
	"context"
	"encoding/json"
	"strconv"
	"time"
)

const (
	KeyAggregationMode      = "aggregation_mode"
	KeyBulkEnabled          = "bulk_enabled"
	KeyBulkMinHealthyLegs   = "bulk_min_healthy_legs"
	KeyBulkMaxLegs          = "bulk_max_legs"
	KeyBulkFallbackMode     = "bulk_fallback_mode"
	KeyBulkRecoverySeconds  = "bulk_recovery_seconds"
	KeyBulkLastStatus       = "bulk_last_status"
	KeyBulkURLAlivePool     = "bulk_url_alive_pool"
	KeyBulkLBStrategy       = "bulk_lb_strategy"
)

const (
	AggregationModeLegacy        = "legacy"
	AggregationModeFlowAggregate = "flow_aggregate"
	BulkFallbackLegacyProxy      = "legacy_proxy"
	BulkLBStickySessions         = "sticky-sessions"
	BulkLBConsistentHash         = "consistent-hashing"
)

// BulkMemberPing last delay test for one PROXY_BULK leg.
type BulkMemberPing struct {
	Tag       string `json:"tag"`
	ProfileID int64  `json:"profile_id,omitempty"`
	Name      string `json:"name,omitempty"`
	DelayMs   int    `json:"delay_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

// BulkStatus runtime snapshot for API/UI.
type BulkStatus struct {
	AggregationMode    string           `json:"aggregation_mode"`
	BulkEnabled        bool             `json:"bulk_enabled"`
	BulkActive         bool             `json:"bulk_active"`
	BulkMemberCount    int              `json:"bulk_member_count"`
	BulkMemberTags     []string         `json:"bulk_member_tags,omitempty"`
	MemberPings        []BulkMemberPing `json:"member_pings,omitempty"`
	BulkMinLegs        int              `json:"bulk_min_legs"`
	BulkMaxLegs        int              `json:"bulk_max_legs"`
	BulkFallbackReason string           `json:"bulk_fallback_reason,omitempty"`
	UpdatedAt          string           `json:"updated_at,omitempty"`
}

func (s *Store) AggregationMode(ctx context.Context) string {
	v, _ := s.GetKV(ctx, KeyAggregationMode)
	if v == AggregationModeFlowAggregate {
		return AggregationModeFlowAggregate
	}
	if v == AggregationModeLegacy {
		return AggregationModeLegacy
	}
	// Unset KV: one tunnel (legacy), no load-balance pool.
	return AggregationModeLegacy
}

func (s *Store) BulkEnabled(ctx context.Context) bool {
	return boolKV(ctx, s, KeyBulkEnabled, false)
}

func (s *Store) BulkMinHealthyLegs(ctx context.Context) int {
	n := intKV(ctx, s, KeyBulkMinHealthyLegs, 2)
	if n < 1 {
		return 1
	}
	return n
}

func (s *Store) BulkMaxLegs(ctx context.Context) int {
	return intKV(ctx, s, KeyBulkMaxLegs, 0)
}

func (s *Store) BulkFallbackMode(ctx context.Context) string {
	v, _ := s.GetKV(ctx, KeyBulkFallbackMode)
	if v == "" {
		return BulkFallbackLegacyProxy
	}
	return v
}

func (s *Store) BulkRecoverySeconds(ctx context.Context) int {
	n := intKV(ctx, s, KeyBulkRecoverySeconds, 60)
	if n < 5 {
		return 5
	}
	return n
}

// BulkLBStrategy returns mihomo load-balance strategy for PROXY_BULK.
func (s *Store) BulkLBStrategy(ctx context.Context) string {
	v, _ := s.GetKV(ctx, KeyBulkLBStrategy)
	if v == BulkLBConsistentHash {
		return BulkLBConsistentHash
	}
	return BulkLBStickySessions
}

// NormalizeBulkLBStrategy maps UI/API values to a supported mihomo strategy name.
func NormalizeBulkLBStrategy(v string) string {
	if v == BulkLBConsistentHash || v == "consistent" {
		return BulkLBConsistentHash
	}
	return BulkLBStickySessions
}

func (s *Store) GetMultipathChannelPool(ctx context.Context) []int64 {
	raw, _ := s.GetKV(ctx, KeyMultipathChannelPool)
	if raw == "" {
		return nil
	}
	return parseIDList(raw)
}

// SetBulkURLAlivePool stores profile IDs with positive URL delay from the latest pretest/warm pass.
func (s *Store) SetBulkURLAlivePool(ctx context.Context, ids []int64) error {
	return s.SetKV(ctx, KeyBulkURLAlivePool, formatIDList(ids))
}

// GetBulkURLAlivePool returns IDs that passed URL delay test on the last selector pass.
func (s *Store) GetBulkURLAlivePool(ctx context.Context) []int64 {
	raw, _ := s.GetKV(ctx, KeyBulkURLAlivePool)
	if raw == "" {
		return nil
	}
	return parseIDList(raw)
}

func (s *Store) SetBulkStatus(ctx context.Context, st BulkStatus) error {
	st.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return s.SetKV(ctx, KeyBulkLastStatus, string(b))
}

func (s *Store) GetBulkStatus(ctx context.Context) BulkStatus {
	raw, _ := s.GetKV(ctx, KeyBulkLastStatus)
	if raw == "" {
		return BulkStatus{
			AggregationMode: s.AggregationMode(ctx),
			BulkEnabled:     s.BulkEnabled(ctx),
			BulkMinLegs:     s.BulkMinHealthyLegs(ctx),
			BulkMaxLegs:     s.EffectiveBulkMaxLegs(ctx),
		}
	}
	var st BulkStatus
	if json.Unmarshal([]byte(raw), &st) != nil {
		return BulkStatus{AggregationMode: s.AggregationMode(ctx)}
	}
	return st
}

func (s *Store) EffectiveBulkMaxLegs(ctx context.Context) int {
	max := s.BulkMaxLegs(ctx)
	if max > 0 {
		return max
	}
	switch s.MultipathPreset(ctx) {
	case "low":
		return 3
	case "high":
		return 8
	default:
		return 6
	}
}

// LoadAggregationSettings reads aggregation-related fields into Settings.
func (s *Store) loadAggregationFields(ctx context.Context, out *Settings) {
	out.AggregationMode = s.AggregationMode(ctx)
	out.BulkEnabled = s.BulkEnabled(ctx)
	out.BulkMinHealthyLegs = s.BulkMinHealthyLegs(ctx)
	out.BulkMaxLegs = s.BulkMaxLegs(ctx)
	out.BulkFallbackMode = s.BulkFallbackMode(ctx)
	out.BulkRecoverySeconds = s.BulkRecoverySeconds(ctx)
	out.BulkLBStrategy = s.BulkLBStrategy(ctx)
}

func (s *Store) saveAggregationFields(ctx context.Context, set Settings) {
	mode := set.AggregationMode
	if mode != AggregationModeFlowAggregate {
		mode = AggregationModeLegacy
	}
	_ = s.SetKV(ctx, KeyAggregationMode, mode)
	_ = s.SetKV(ctx, KeyBulkEnabled, boolStr(set.BulkEnabled))
	minLegs := set.BulkMinHealthyLegs
	if minLegs < 1 {
		minLegs = 2
	}
	_ = s.SetKV(ctx, KeyBulkMinHealthyLegs, strconv.Itoa(minLegs))
	_ = s.SetKV(ctx, KeyBulkMaxLegs, strconv.Itoa(set.BulkMaxLegs))
	fb := set.BulkFallbackMode
	if fb == "" {
		fb = BulkFallbackLegacyProxy
	}
	_ = s.SetKV(ctx, KeyBulkFallbackMode, fb)
	rec := set.BulkRecoverySeconds
	if rec < 5 {
		rec = 60
	}
	_ = s.SetKV(ctx, KeyBulkRecoverySeconds, strconv.Itoa(rec))
	_ = s.SetKV(ctx, KeyBulkLBStrategy, NormalizeBulkLBStrategy(set.BulkLBStrategy))
}
