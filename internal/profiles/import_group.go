package profiles

import (
	"context"
	"fmt"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

// ImportLinesToGroup upserts URIs into a specific group (manual collection or subscription refresh).
func ImportLinesToGroup(ctx context.Context, st *store.Store, groupID int64, lines []string) ([]ImportResult, error) {
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
		id, err := st.UpsertProfileInGroup(ctx, groupID, name, typ, line, int64(1000+i), false, false)
		if err != nil {
			return out, err
		}
		out = append(out, ImportResult{ID: id, Name: name, Type: typ})
	}
	return out, nil
}
