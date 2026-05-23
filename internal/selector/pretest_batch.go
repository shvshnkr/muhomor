package selector

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/mihomo"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/store"
)

const pretestMicroAPIWait = 18 * time.Second
const pretestMicroAPIWaitWSL = 28 * time.Second

func pretestMicroStartWait() time.Duration {
	if runtime.GOOS == "linux" && paths.IsWSL() {
		return pretestMicroAPIWaitWSL
	}
	return pretestMicroAPIWait
}

func (e *EphemeralTester) testProfilesMicroBatchOnce(ctx context.Context, profiles []store.Profile) map[int64]int {
	if len(profiles) == 0 {
		return nil
	}

	testURL := ""
	perProxyMs := 8000
	if e.Store != nil {
		testURL = e.Store.ConnectionTestURL(ctx)
		if t := e.Store.ConnectionTestTimeoutMs(ctx); t > 0 {
			perProxyMs = t
		}
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
		Secret:             "pretest",
		Mode:               "rule",
		LogLevel:           "error",
	}
	yaml, idToName, err := configgen.BuildPretestBatch(profiles, opt, testURL, perProxyMs)
	if err != nil || len(idToName) == 0 {
		return nil
	}
	dir, err := os.MkdirTemp("", "muhomor-pretest-micro-*")
	if err != nil {
		return nil
	}
	defer os.RemoveAll(dir)
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		return nil
	}

	bin := e.mihomoBinFast()
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
		e.logPretestWarn("pretest micro-batch start failed", "err", err, "proxies", len(idToName))
		return nil
	}
	defer client.Stop()

	groupTimeoutMs := mihomo.GroupDelayAPITimeout(perProxyMs, len(idToName))
	httpWait := time.Duration(groupTimeoutMs)*time.Millisecond + 15*time.Second
	batchCtx, batchCancel := context.WithTimeout(ctx, httpWait)
	defer batchCancel()

	delays, err := client.GroupDelay(batchCtx, configgen.PretestGroupName, testURL, groupTimeoutMs)
	if err != nil {
		e.logPretestWarn("pretest micro-batch delay failed", "err", err, "proxies", len(idToName))
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

func (e *EphemeralTester) logPretestWarn(msg string, args ...any) {
	if e.Log != nil {
		kv := append([]any{"event", "H4-batch"}, args...)
		e.Log.Warn(msg, kv...)
	}
}

func (e *EphemeralTester) savePretestDiag(cfgPath, yaml string) {
	var bases []string
	if b := os.Getenv("MUHOMOR_DATA_DIR"); b != "" {
		bases = append(bases, b)
	}
	if e.Store != nil {
		if wd, err := os.Getwd(); err == nil {
			bases = append(bases, filepath.Join(wd, "data"))
		}
	}
	seen := map[string]struct{}{}
	for _, base := range bases {
		if base == "" {
			continue
		}
		if _, ok := seen[base]; ok {
			continue
		}
		seen[base] = struct{}{}
		cache := filepath.Join(base, "cache")
		if err := os.MkdirAll(cache, 0o700); err != nil {
			continue
		}
		_ = os.WriteFile(filepath.Join(cache, "pretest-last.yaml"), []byte(yaml), 0o600)
		_ = os.WriteFile(filepath.Join(cache, "pretest-last-path.txt"), []byte(cfgPath+"\n"), 0o600)
	}
}
