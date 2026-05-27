package mihomo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/store"
)

const (
	pickerControllerPort = 8760
	pickerMixedPort      = 2182
	pickerReloadDebounce = 30 * time.Second
)

// Picker is a long-lived mihomo subprocess for batch URL-test (/group/.../delay).
type Picker struct {
	mu          sync.Mutex
	client      *Client
	configDir   string
	configPath  string
	binPath     string
	profilesKey string
	lastReload  time.Time
	running     bool
	Log         *slog.Logger
}

// NewPicker creates a picker bound to layout.RuntimeDir/picker.
func NewPicker(layout paths.Layout, binPath string) *Picker {
	dir := filepath.Join(layout.RuntimeDir, "picker")
	if binPath == "" {
		binPath = ResolveBin()
	}
	return &Picker{
		configDir:  dir,
		configPath: filepath.Join(dir, "config.yaml"),
		binPath:    binPath,
	}
}

// Available reports whether the picker subprocess is running.
func (p *Picker) Available() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running && p.client != nil
}

func pickerSecret() string { return "picker" }

func probePickerAPI(ctx context.Context, ctl, secret string) *Client {
	c := NewClient(ClientOptions{
		Controller:      ctl,
		Secret:          secret,
		APIReadyTimeout: 2 * time.Second,
	})
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if _, err := c.Version(probeCtx); err != nil {
		return nil
	}
	return c
}

func isPickerBindError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "bind") && strings.Contains(msg, "already in use") ||
		strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage")
}

func (p *Picker) logRecover(action string, err error) {
	if p.Log == nil {
		return
	}
	if err != nil {
		p.Log.Info("picker ensure", "picker_recover", action, "err", err)
		return
	}
	p.Log.Info("picker ensure", "picker_recover", action)
}

// installReusedClientLocked attaches to a live picker API; caller must hold p.mu.
func (p *Picker) installReusedClientLocked(c *Client) {
	if p.client != nil && p.client.cmd != nil {
		p.client.Stop()
	}
	p.client = c
	p.running = true
	if p.profilesKey == "" {
		p.profilesKey = "bootstrap"
	}
	p.logRecover("reuse", nil)
}

func (p *Picker) attachExisting(ctx context.Context, ctl, secret string) bool {
	c := probePickerAPI(ctx, ctl, secret)
	if c == nil {
		return false
	}
	p.mu.Lock()
	p.installReusedClientLocked(c)
	p.mu.Unlock()
	return true
}

// Ensure starts the picker subprocess if needed (bootstrap config, then reload on batch).
func (p *Picker) Ensure(ctx context.Context) error {
	p.mu.Lock()
	if p.running && p.client != nil {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()
	if err := os.MkdirAll(p.configDir, 0o700); err != nil {
		return err
	}
	ctl := paths.MihomoControllerAddr(pickerControllerPort)
	secret := pickerSecret()
	if p.attachExisting(ctx, ctl, secret) {
		return nil
	}
	opt := configgen.BuildOptions{
		MixedPort:          pickerMixedPort,
		ExternalController: ctl,
		Secret:             secret,
		Mode:               "rule",
		LogLevel:           "error",
	}
	yaml := configgen.BuildPickerBootstrap(opt)
	if err := os.WriteFile(p.configPath, []byte(yaml), 0o600); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running && p.client != nil {
		return nil
	}
	if p.client != nil {
		p.client.Stop()
		p.client = nil
		p.running = false
	}
	p.client = NewClient(ClientOptions{
		BinPath:         p.binPath,
		ConfigPath:      p.configPath,
		ConfigDir:       p.configDir,
		Controller:      ctl,
		Secret:          secret,
		APIReadyTimeout: 18 * time.Second,
	})
	startCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := p.client.Start(startCtx); err != nil {
		if isPickerBindError(err) {
			if c := probePickerAPI(ctx, ctl, secret); c != nil {
				p.installReusedClientLocked(c)
				p.profilesKey = "bootstrap"
				p.lastReload = time.Now()
				return nil
			}
		}
		p.client = nil
		p.running = false
		return fmt.Errorf("picker ensure: %w", err)
	}
	p.running = true
	p.profilesKey = "bootstrap"
	p.lastReload = time.Now()
	p.logRecover("restart", nil)
	return nil
}

// Stop shuts down the picker subprocess.
func (p *Picker) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client != nil {
		p.client.Stop()
		p.client = nil
	}
	p.running = false
}

const pickerBatchCap = 12

// GroupDelayProfiles reloads batch config and returns profile_id -> delay_ms.
func (p *Picker) GroupDelayProfiles(ctx context.Context, profiles []store.Profile, st *store.Store) map[int64]int {
	if len(profiles) == 0 {
		return nil
	}
	if len(profiles) > pickerBatchCap {
		profiles = profiles[:pickerBatchCap]
	}
	testURL, perProxyMs, idToName, err := p.prepareBatch(ctx, profiles, st)
	if err != nil || len(idToName) == 0 {
		return nil
	}
	if err := p.reloadProfiles(ctx, profiles, st, testURL, perProxyMs); err != nil {
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
	delays, err := client.GroupDelay(batchCtx, configgen.PretestGroupName, testURL, groupTimeoutMs)
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

func (p *Picker) prepareBatch(ctx context.Context, profiles []store.Profile, st *store.Store) (testURL string, perProxyMs int, idToName map[int64]string, err error) {
	perProxyMs = 8000
	if st != nil {
		testURL = st.ConnectionTestURL(ctx)
		if t := st.ConnectionTestTimeoutMs(ctx); t > 0 {
			perProxyMs = t
		}
	}
	ctl := paths.MihomoControllerAddr(pickerControllerPort)
	opt := configgen.BuildOptions{
		MixedPort:          pickerMixedPort,
		ExternalController: ctl,
		Secret:             "picker",
		Mode:               "rule",
		LogLevel:           "error",
	}
	_, idToName, err = configgen.BuildPretestBatch(profiles, opt, testURL, perProxyMs)
	return testURL, perProxyMs, idToName, err
}

func (p *Picker) reloadProfiles(ctx context.Context, profiles []store.Profile, st *store.Store, testURL string, perProxyMs int) error {
	key := profilesKey(profiles)
	p.mu.Lock()
	needStart := !p.running || p.client == nil
	debounce := time.Since(p.lastReload) < pickerReloadDebounce
	sameKey := p.profilesKey == key
	p.mu.Unlock()

	if err := os.MkdirAll(p.configDir, 0o700); err != nil {
		return err
	}
	ctl := paths.MihomoControllerAddr(pickerControllerPort)
	opt := configgen.BuildOptions{
		MixedPort:          pickerMixedPort,
		ExternalController: ctl,
		Secret:             "picker",
		Mode:               "rule",
		LogLevel:           "error",
	}
	yaml, _, err := configgen.BuildPretestBatch(profiles, opt, testURL, perProxyMs)
	if err != nil {
		return err
	}
	if err := os.WriteFile(p.configPath, []byte(yaml), 0o600); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if needStart {
		if c := probePickerAPI(ctx, ctl, opt.Secret); c != nil {
			p.installReusedClientLocked(c)
			p.profilesKey = key
			p.lastReload = time.Now()
			return nil
		}
		if p.client != nil {
			p.client.Stop()
			p.client = nil
			p.running = false
		}
		p.client = NewClient(ClientOptions{
			BinPath:         p.binPath,
			ConfigPath:      p.configPath,
			ConfigDir:       p.configDir,
			Controller:      ctl,
			Secret:          opt.Secret,
			APIReadyTimeout: 18 * time.Second,
		})
		startCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		if err := p.client.Start(startCtx); err != nil {
			if isPickerBindError(err) {
				if c := probePickerAPI(ctx, ctl, opt.Secret); c != nil {
					p.installReusedClientLocked(c)
					p.profilesKey = key
					p.lastReload = time.Now()
					return nil
				}
			}
			p.client = nil
			p.running = false
			return fmt.Errorf("picker start: %w", err)
		}
		p.running = true
		p.profilesKey = key
		p.lastReload = time.Now()
		p.logRecover("restart", nil)
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
			Secret:          opt.Secret,
			APIReadyTimeout: 18 * time.Second,
		})
		if err := p.client.Start(startCtx); err != nil {
			if isPickerBindError(err) {
				if c := probePickerAPI(ctx, ctl, opt.Secret); c != nil {
					p.installReusedClientLocked(c)
					p.profilesKey = key
					p.lastReload = time.Now()
					return nil
				}
			}
			p.client = nil
			return err
		}
		p.logRecover("restart", nil)
	}
	p.profilesKey = key
	p.lastReload = time.Now()
	return nil
}

func profilesKey(profiles []store.Profile) string {
	ids := make([]int64, len(profiles))
	for i, p := range profiles {
		ids[i] = p.ID
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	h := sha256.New()
	for _, id := range ids {
		fmt.Fprintf(h, "%d,", id)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}
