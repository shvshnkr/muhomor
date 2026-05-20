package profiles

import (
	"context"
	"fmt"
	"io"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

// ImportLines upserts supported proxy URIs; returns per-line results.
func ImportLines(ctx context.Context, st *store.Store, lines []string) ([]ImportResult, error) {
	var out []ImportResult
	for i, line := range lines {
		line = trimLine(line)
		if line == "" {
			continue
		}
		if reason := subscription.UnsupportedReason(line); reason != "" {
			out = append(out, ImportResult{Skip: true, Reason: reason, Line: truncate(line, 60)})
			continue
		}
		typ := subscription.Scheme(line)
		name := fmt.Sprintf("imported-%d", i+1)
		if typ == "vless" {
			if p, err := configgen.ParseVLESSURI(line); err == nil && p.Name != "" {
				name = p.Name
			}
		}
		id, err := st.UpsertProfile(ctx, name, typ, line)
		if err != nil {
			return out, err
		}
		out = append(out, ImportResult{ID: id, Name: name, Type: typ})
	}
	return out, nil
}

// ImportReader parses subscription text from r.
func ImportReader(ctx context.Context, st *store.Store, r io.Reader) ([]ImportResult, error) {
	lines, err := subscription.ParseLines(r)
	if err != nil {
		return nil, err
	}
	return ImportLines(ctx, st, lines)
}

// ImportResult is one import outcome.
type ImportResult struct {
	ID     int64  `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
	Skip   bool   `json:"skip,omitempty"`
	Reason string `json:"reason,omitempty"`
	Line   string `json:"line,omitempty"`
}

func trimLine(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\r' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
