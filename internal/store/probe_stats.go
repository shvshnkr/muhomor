package store

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// ProbeStats is persisted JSON in KeyProbeSchedulerStats and exposed via API.
type ProbeStats struct {
	TotalEnabled       int    `json:"total_enabled"`
	Unknown            int    `json:"unknown"`
	Candidate          int    `json:"candidate"`
	Alive              int    `json:"alive"`
	Suspect            int    `json:"suspect"`
	Dead               int    `json:"dead"`
	Cemetery           int    `json:"cemetery"`
	LastTickChecked    int    `json:"last_tick_checked"`
	LastTickOK         int    `json:"last_tick_ok"`
	LastTickFail       int    `json:"last_tick_fail"`
	SchedulerEnabled   bool   `json:"scheduler_enabled"`
	WarmSelectEnabled  bool   `json:"warm_select_enabled"`
	Preset             string `json:"preset,omitempty"`
	LastSelectReason   string `json:"last_select_reason,omitempty"`
	UpdatedAt          string `json:"updated_at,omitempty"`
}

func (s *Store) SetProbeStats(ctx context.Context, st ProbeStats) error {
	st.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return s.SetKV(ctx, KeyProbeSchedulerStats, string(b))
}

func (s *Store) GetProbeStats(ctx context.Context) ProbeStats {
	raw, _ := s.GetKV(ctx, KeyProbeSchedulerStats)
	if raw == "" {
		return s.probeStatsSkeleton(ctx)
	}
	var st ProbeStats
	if json.Unmarshal([]byte(raw), &st) == nil {
		st.SchedulerEnabled = s.ProbeSchedulerEnabled(ctx)
		st.WarmSelectEnabled = s.ProbeWarmSelectEnabled(ctx)
		if st.Preset == "" {
			st.Preset = s.ProbePreset(ctx)
		}
		if st.LastSelectReason == "" {
			st.LastSelectReason, _ = s.GetKV(ctx, KeyProbeLastSelectReason)
		}
		if st.TotalEnabled == 0 {
			if n, err := s.CountEnabledProfiles(ctx); err == nil {
				st.TotalEnabled = n
			}
		}
		return st
	}
	return parseLegacyProbeStats(raw, s.probeStatsSkeleton(ctx))
}

func (s *Store) probeStatsSkeleton(ctx context.Context) ProbeStats {
	st := ProbeStats{
		SchedulerEnabled:  s.ProbeSchedulerEnabled(ctx),
		WarmSelectEnabled: s.ProbeWarmSelectEnabled(ctx),
		Preset:            s.ProbePreset(ctx),
	}
	st.LastSelectReason, _ = s.GetKV(ctx, KeyProbeLastSelectReason)
	counts, err := s.CountProbeStates(ctx)
	if err != nil {
		return st
	}
	st.Unknown = counts[ProbeUnknown]
	st.Candidate = counts[ProbeCandidate]
	st.Alive = counts[ProbeAlive]
	st.Suspect = counts[ProbeSuspect]
	st.Dead = counts[ProbeDead]
	st.Cemetery = counts[ProbeCemetery]
	if n, err := s.CountEnabledProfiles(ctx); err == nil {
		st.TotalEnabled = n
	}
	return st
}

func parseLegacyProbeStats(raw string, base ProbeStats) ProbeStats {
	for _, part := range strings.Split(raw, " ") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		n, _ := strconv.Atoi(kv[1])
		switch kv[0] {
		case "checked":
			base.LastTickChecked = n
		case "ok":
			base.LastTickOK = n
		case "fail":
			base.LastTickFail = n
		case "alive":
			base.Alive = n
		case "dead":
			base.Dead = n
		case "cemetery":
			base.Cemetery = n
		}
	}
	return base
}

func (s *Store) SetLastSelectReason(ctx context.Context, reason string) error {
	return s.SetKV(ctx, KeyProbeLastSelectReason, reason)
}

func (s *Store) CountEnabledProfiles(ctx context.Context) (int, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM profiles WHERE enabled = 1`)
	var n int
	return n, row.Scan(&n)
}
