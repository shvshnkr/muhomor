package selector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/mihomo"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/probe/blexit"
	"github.com/muhomor/muhomor/internal/store"
)

// EphemeralSuiteRunner runs BL suite in a short-lived mihomo subprocess.
type EphemeralSuiteRunner struct {
	MihomoBin string
}

func (r *EphemeralSuiteRunner) Name() string { return "ephemeral" }

func (r *EphemeralSuiteRunner) RunSuite(ctx context.Context, profiles []store.Profile, st *store.Store, urls []string, minPass int) (map[int64]bool, error) {
	if len(profiles) == 0 || len(urls) == 0 {
		return nil, nil
	}
	if len(profiles) > blexit.TestCap {
		profiles = profiles[:blexit.TestCap]
	}
	perProxyMs := 8000
	if st != nil {
		perProxyMs = st.BLExitTimeoutMs(ctx)
	}
	port := atomic.AddUint32(&pretestPort, 1)
	if port > 65000 {
		atomic.StoreUint32(&pretestPort, 9000)
		port = atomic.AddUint32(&pretestPort, 1)
	}
	ctl := paths.MihomoControllerAddr(int(port))
	opt := configgen.BuildOptions{
		MixedPort:          17890 + int(port%1000),
		ExternalController: ctl,
		Secret:             "pretest-bl",
		Mode:               "rule",
		LogLevel:           "error",
	}
	yaml, groups, idToName, err := configgen.BuildPretestBLMulti(profiles, opt, urls, perProxyMs)
	if err != nil || len(groups) == 0 || len(idToName) == 0 {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "muhomor-pretest-bl-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		return nil, err
	}
	bin := r.MihomoBin
	if bin == "" {
		bin = mihomo.ResolveBin()
	}
	apiWait := pretestMicroStartWait()
	client := mihomo.NewClient(mihomo.ClientOptions{
		BinPath:         bin,
		ConfigPath:      cfgPath,
		ConfigDir:       dir,
		Controller:      ctl,
		Secret:          opt.Secret,
		APIReadyTimeout: apiWait,
	})
	startCtx, cancel := context.WithTimeout(ctx, apiWait)
	defer cancel()
	if err := client.Start(startCtx); err != nil {
		return nil, err
	}
	defer client.Stop()

	nameToID := make(map[string]int64, len(idToName))
	for id, name := range idToName {
		nameToID[name] = id
	}
	groupTimeoutMs := mihomo.GroupDelayAPITimeout(perProxyMs, len(profiles))
	httpWait := time.Duration(groupTimeoutMs)*time.Millisecond + 15*time.Second
	var perURL []map[int64]int
	for _, gname := range groups {
		testURL := blTestURLForGroup(urls, gname)
		if testURL == "" {
			continue
		}
		batchCtx, batchCancel := context.WithTimeout(ctx, httpWait)
		delays, err := client.GroupDelay(batchCtx, gname, testURL, groupTimeoutMs)
		batchCancel()
		if err != nil || len(delays) == 0 {
			continue
		}
		m := make(map[int64]int, len(delays))
		for name, ms := range delays {
			if id, ok := nameToID[name]; ok && ms > 0 {
				m[id] = ms
			}
		}
		if len(m) > 0 {
			perURL = append(perURL, m)
		}
	}
	if len(perURL) == 0 {
		return nil, nil
	}
	return blexit.MergeSuite(perURL, minPass), nil
}

func blTestURLForGroup(urls []string, groupName string) string {
	for i, u := range urls {
		if blGroupNameForURL(u, i) == groupName {
			return u
		}
	}
	return ""
}

func blGroupNameForURL(u string, idx int) string {
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

var _ blexit.SuiteRunner = (*EphemeralSuiteRunner)(nil)
