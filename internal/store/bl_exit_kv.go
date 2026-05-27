package store

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

const blExitOKTTL = 20 * time.Minute

var defaultBLExitTestURLs = []string{
	"https://api.telegram.org/",
	"https://web.whatsapp.com/",
}

const (
	BLExitScopeRuExit = "ru_exit"
	BLExitScopeAll    = "all"
)

// BLExitFilterEnabled is on by default (open-network pre-connect BL filter).
func (s *Store) BLExitFilterEnabled(ctx context.Context) bool {
	v, _ := s.GetKV(ctx, KeyProbeBLExitFilterEnabled)
	if v == "" {
		return true
	}
	return v == "1" || strings.EqualFold(v, "true")
}

// BLExitFilterScope returns ru_exit (default) or all.
func (s *Store) BLExitFilterScope(ctx context.Context) string {
	v, _ := s.GetKV(ctx, KeyProbeBLExitScope)
	v = strings.ToLower(strings.TrimSpace(v))
	if v == BLExitScopeAll {
		return BLExitScopeAll
	}
	return BLExitScopeRuExit
}

// BLExitFilterAllProfiles is true when BL filter applies to every proxy (not only ru_exit).
func (s *Store) BLExitFilterAllProfiles(ctx context.Context) bool {
	return s.BLExitFilterScope(ctx) == BLExitScopeAll
}

func (s *Store) BLExitTestURLs(ctx context.Context) []string {
	raw, _ := s.GetKV(ctx, KeyProbeBLExitTestURLs)
	if raw == "" {
		return append([]string(nil), defaultBLExitTestURLs...)
	}
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "[") {
		var urls []string
		if json.Unmarshal([]byte(raw), &urls) == nil && len(urls) > 0 {
			return urls
		}
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return append([]string(nil), defaultBLExitTestURLs...)
	}
	return out
}

func (s *Store) BLExitTimeoutMs(ctx context.Context) int {
	v, _ := s.GetKV(ctx, KeyProbeBLExitTimeoutMs)
	if v == "" {
		return 8000
	}
	n, _ := strconv.Atoi(v)
	if n <= 0 {
		return 8000
	}
	return n
}

func (s *Store) BLExitMinPass(ctx context.Context) int {
	v, _ := s.GetKV(ctx, KeyProbeBLExitMinPass)
	if v == "" {
		return 1
	}
	n, _ := strconv.Atoi(v)
	if n <= 0 {
		return 1
	}
	return n
}

// SetBLExitOK records a fresh BL pass for standby fast-path.
func (s *Store) SetBLExitOK(ctx context.Context, profileID int64, delayMs int) error {
	if profileID <= 0 || delayMs <= 0 {
		return nil
	}
	val := strconv.Itoa(delayMs) + "@" + time.Now().UTC().Format(time.RFC3339)
	return s.SetKV(ctx, KeyBLExitOK(profileID), val)
}

// BLExitOK returns cached BL delay if fresh (within blExitOKTTL).
func (s *Store) BLExitOK(ctx context.Context, profileID int64) (delayMs int, ok bool) {
	raw, _ := s.GetKV(ctx, KeyBLExitOK(profileID))
	if raw == "" {
		return 0, false
	}
	at := strings.LastIndex(raw, "@")
	if at < 0 {
		return 0, false
	}
	ms, err := strconv.Atoi(raw[:at])
	if err != nil || ms <= 0 {
		return 0, false
	}
	ts, err := time.Parse(time.RFC3339, raw[at+1:])
	if err != nil || time.Since(ts) > blExitOKTTL {
		return 0, false
	}
	return ms, true
}

// SimpleModeWLOnly reads KV set at connect (open vs WL pool).
func (s *Store) SimpleModeWLOnly(ctx context.Context) bool {
	return boolKV(ctx, s, KeySimpleModeUseWLPoolOnly, false)
}
