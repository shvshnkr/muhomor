package mihomo

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/probe/blexit"
	"github.com/muhomor/muhomor/internal/store"
)

// GroupDelayProfilesForURL runs batch delay against a custom test URL (BL suite member).
func (p *Picker) GroupDelayProfilesForURL(ctx context.Context, profiles []store.Profile, st *store.Store, testURL, groupName string) map[int64]int {
	if len(profiles) == 0 || testURL == "" {
		return nil
	}
	if len(profiles) > pickerBatchCap {
		profiles = profiles[:pickerBatchCap]
	}
	perProxyMs := 8000
	if st != nil {
		perProxyMs = st.BLExitTimeoutMs(ctx)
	}
	ctl := paths.MihomoControllerAddr(pickerControllerPort)
	opt := configgen.BuildOptions{
		MixedPort:          pickerMixedPort,
		ExternalController: ctl,
		Secret:             "picker",
		Mode:               "rule",
		LogLevel:           "error",
	}
	yaml, idToName, err := configgen.BuildPretestBLBatch(profiles, opt, testURL, perProxyMs, groupName)
	if err != nil || len(idToName) == 0 {
		return nil
	}
	key := profilesKey(profiles) + ":" + groupName
	if err := p.reloadYAML(ctx, yaml, key); err != nil {
		return nil
	}
	p.mu.Lock()
	client := p.client
	p.mu.Unlock()
	if client == nil {
		return nil
	}
	groupTimeoutMs := GroupDelayAPITimeout(perProxyMs, len(profiles))
	httpWait := time.Duration(groupTimeoutMs)*time.Millisecond + 15*time.Second
	batchCtx, cancel := context.WithTimeout(ctx, httpWait)
	defer cancel()
	delays, err := client.GroupDelay(batchCtx, groupName, testURL, groupTimeoutMs)
	if err != nil {
		return nil
	}
	nameToID := make(map[string]int64, len(idToName))
	for id, name := range idToName {
		nameToID[name] = id
	}
	out := make(map[int64]int, len(delays))
	for name, ms := range delays {
		if id, ok := nameToID[name]; ok && ms > 0 {
			out[id] = ms
		}
	}
	return out
}

// GroupDelayProfilesBLSuite OR-merges delays across BL test URLs.
func (p *Picker) GroupDelayProfilesBLSuite(ctx context.Context, profiles []store.Profile, st *store.Store, urls []string, minPass int) map[int64]bool {
	if len(urls) == 0 {
		return nil
	}
	var perURL []map[int64]int
	for i, u := range urls {
		gname := blPickerGroupName(u, i)
		m := p.GroupDelayProfilesForURL(ctx, profiles, st, u, gname)
		if len(m) > 0 {
			perURL = append(perURL, m)
		}
	}
	if len(perURL) == 0 {
		return nil
	}
	return blexit.MergeSuite(perURL, minPass)
}

func blPickerGroupName(u string, idx int) string {
	lu := strings.ToLower(u)
	switch {
	case strings.Contains(lu, "telegram"):
		return configgen.PretestBLGroupTG
	case strings.Contains(lu, "whatsapp"):
		return configgen.PretestBLGroupWA
	default:
		return fmt.Sprintf("MUHOMOR_BL_%d", idx)
	}
}

func (p *Picker) reloadYAML(ctx context.Context, yaml, key string) error {
	if err := os.MkdirAll(p.configDir, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(p.configPath, []byte(yaml), 0o600); err != nil {
		return err
	}
	ctl := paths.MihomoControllerAddr(pickerControllerPort)
	p.mu.Lock()
	defer p.mu.Unlock()
	needStart := !p.running || p.client == nil
	debounce := time.Since(p.lastReload) < pickerReloadDebounce
	sameKey := p.profilesKey == key
	if needStart {
		p.client = NewClient(ClientOptions{
			BinPath:         p.binPath,
			ConfigPath:      p.configPath,
			ConfigDir:       p.configDir,
			Controller:      ctl,
			Secret:          "picker",
			APIReadyTimeout: 18 * time.Second,
		})
		startCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		if err := p.client.Start(startCtx); err != nil {
			p.client = nil
			p.running = false
			return fmt.Errorf("picker start: %w", err)
		}
		p.running = true
		p.profilesKey = key
		p.lastReload = time.Now()
		return nil
	}
	if debounce && sameKey {
		return nil
	}
	reloadCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := p.client.Reload(reloadCtx); err != nil {
		p.client.Stop()
		p.client = nil
		p.running = false
		startCtx, startCancel := context.WithTimeout(ctx, 20*time.Second)
		defer startCancel()
		p.client = NewClient(ClientOptions{
			BinPath:         p.binPath,
			ConfigPath:      p.configPath,
			ConfigDir:       p.configDir,
			Controller:      ctl,
			Secret:          "picker",
			APIReadyTimeout: 18 * time.Second,
		})
		if err := p.client.Start(startCtx); err != nil {
			p.client = nil
			return err
		}
		p.running = true
	}
	p.profilesKey = key
	p.lastReload = time.Now()
	return nil
}
