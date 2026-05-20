package subscription

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/store"
)

// Updater fetches subscription groups (HTTP + User-Agent per group).
type Updater struct {
	Store     *store.Store
	Client    *http.Client
	UserAgent func(ctx context.Context, groupID int64) string
}

func (u *Updater) RefreshDue(ctx context.Context, internetOK bool) error {
	if !internetOK {
		return fmt.Errorf("subscription refresh skipped: no internet")
	}
	groups, err := u.Store.ListGroups(ctx)
	if err != nil {
		return err
	}
	var lastErr error
	for _, g := range groups {
		if g.Kind != store.GroupKindSubscription || g.SubscriptionLink == "" {
			continue
		}
		if _, err := u.RefreshGroup(ctx, g.ID); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// RefreshGroup downloads subscription for one group and upserts profiles into it.
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
	ua := ""
	if u.UserAgent != nil {
		ua = u.UserAgent(ctx, groupID)
	}
	lines, err := u.fetchSubscription(ctx, g.SubscriptionLink, ua)
	if err != nil {
		return 0, err
	}
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
		if _, err := u.Store.UpsertProfileInGroup(ctx, groupID, name, typ, line, int64(1000+i), false); err != nil {
			return added, err
		}
		added++
	}
	_ = u.Store.TouchGroupUpdated(ctx, groupID)
	return added, nil
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
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("subscription HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	// Plain text or base64-ish body — ParseLines handles lines.
	if strings.HasPrefix(strings.TrimSpace(string(body)), "{") {
		// Some subs return JSON; try line-by-line anyway.
	}
	return ParseLines(strings.NewReader(string(body)))
}

// FetchGroup is deprecated; use RefreshGroup.
func (u *Updater) FetchGroup(ctx context.Context, groupID int64, link string) error {
	_ = link
	_, err := u.RefreshGroup(ctx, groupID)
	return err
}
