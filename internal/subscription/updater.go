package subscription

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/profileclass"
	"github.com/muhomor/muhomor/internal/store"
)

// Updater fetches subscription groups; learns User-Agent per group (Dahusim fetch profiles).
type Updater struct {
	Store  *store.Store
	Client *http.Client
}

// RefreshGroup downloads subscription and upserts profiles; auto-picks and saves User-Agent.
func (u *Updater) RefreshGroup(ctx context.Context, groupID int64) (added int, err error) {
	g, err := u.groupByID(ctx, groupID)
	if err != nil {
		return 0, err
	}
	if g.Kind != store.GroupKindSubscription {
		return 0, fmt.Errorf("group is not a subscription list")
	}
	if g.SubscriptionLink == "" {
		return 0, fmt.Errorf("subscription link empty")
	}
	lines, usedUA, err := u.fetchWithLearnedUA(ctx, groupID, g.SubscriptionLink)
	if err != nil {
		return 0, err
	}
	if len(lines) > maxLinesPerGroup {
		lines = lines[:maxLinesPerGroup]
	}
	wlMarked := IsWhiteBoltWLGroup(g.Name)
	keepURIs := make(map[string]struct{}, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if reason := UnsupportedReason(line); reason != "" {
			continue
		}
		typ := Scheme(line)
		name := fmt.Sprintf("sub-%d-%d", groupID, i+1)
		if typ == "vless" {
			if p, err := configgen.ParseVLESSURI(line); err == nil && p.Name != "" {
				name = p.Name
			}
		}
		ruExit := profileclass.RuExitMarkedByName(name)
		if _, err := u.Store.UpsertProfileInGroup(ctx, groupID, name, typ, line, int64(1000+i), false, wlMarked, ruExit); err != nil {
			return added, err
		}
		keepURIs[line] = struct{}{}
		added++
	}
	if _, err := u.Store.PruneGroupProfilesNotInURIs(ctx, groupID, keepURIs); err != nil {
		return added, err
	}
	if _, err := RepairTruncatedURIs(ctx, u.Store); err != nil {
		return added, err
	}
	_ = u.Store.TouchGroupUpdated(ctx, groupID)
	_ = usedUA
	return added, nil
}

func (u *Updater) fetchWithLearnedUA(ctx context.Context, groupID int64, link string) ([]string, string, error) {
	saved, _ := u.Store.GetKV(ctx, store.KeyGroupUserAgent(groupID))
	var lastErr error
	for _, ua := range ProbeUserAgentCandidates(link, saved) {
		lines, err := u.fetchSubscription(ctx, link, ua)
		if err != nil {
			lastErr = err
			continue
		}
		_ = u.Store.SetKV(ctx, store.KeyGroupUserAgent(groupID), ua)
		return lines, ua, nil
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", fmt.Errorf("subscription fetch failed for %s", link)
}

func (u *Updater) groupByID(ctx context.Context, id int64) (store.Group, error) {
	groups, err := u.Store.ListGroups(ctx)
	if err != nil {
		return store.Group{}, err
	}
	for _, g := range groups {
		if g.ID == id {
			return g, nil
		}
	}
	return store.Group{}, fmt.Errorf("group %d not found", id)
}

func (u *Updater) fetchSubscription(ctx context.Context, link, userAgent string) ([]string, error) {
	client := u.Client
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", strings.TrimSpace(userAgent))
	req.Header.Set("Accept", "*/*")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", link, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 120 {
			snippet = snippet[:120] + "…"
		}
		return nil, fmt.Errorf("subscription HTTP %d from %s (ua=%s; body: %q)", resp.StatusCode, link, UAModeLabel(userAgent), snippet)
	}
	body = NormalizeSubscriptionBody(body)
	lines, err := ParseLines(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("subscription empty or unsupported format from %s", link)
	}
	return lines, nil
}

// FetchGroup is deprecated; use RefreshGroup.
func (u *Updater) FetchGroup(ctx context.Context, groupID int64, link string) error {
	_ = link
	_, err := u.RefreshGroup(ctx, groupID)
	return err
}
