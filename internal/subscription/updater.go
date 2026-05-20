package subscription

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/muhomor/muhomor/internal/store"
)

// Updater fetches subscription groups (offline guard: needs live internet).
type Updater struct {
	Store  *store.Store
	Client *http.Client
}

func (u *Updater) RefreshDue(ctx context.Context, internetOK bool) error {
	if !internetOK {
		return fmt.Errorf("subscription refresh skipped: no internet")
	}
	groups, err := u.Store.ListGroups(ctx)
	if err != nil {
		return err
	}
	for _, g := range groups {
		if g.SubscriptionLink == "" {
			continue
		}
		if err := u.FetchGroup(ctx, g.ID, g.SubscriptionLink); err != nil {
			continue
		}
	}
	return nil
}

func (u *Updater) FetchGroup(ctx context.Context, groupID int64, link string) error {
	if u.Client == nil {
		u.Client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return err
	}
	resp, err := u.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	lines, err := ParseVLESSLines(strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	for i, line := range lines {
		if reason := UnsupportedReason(line); reason != "" {
			continue
		}
		typ := "vless"
		if strings.HasPrefix(line, "trojan://") {
			typ = "trojan"
		} else if strings.HasPrefix(line, "hysteria2://") || strings.HasPrefix(line, "hysteria://") {
			typ = "hysteria2"
		}
		name := fmt.Sprintf("sub-%d-%d", groupID, i+1)
		_, _ = u.Store.UpsertProfile(ctx, name, typ, line)
	}
	return nil
}
