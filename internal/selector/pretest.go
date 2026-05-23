package selector

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/mihomo"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/store"
)

var pretestPort uint32 = 9000

// EphemeralTester runs url-test via short-lived mihomo (pre-connect, like libcore forTest).
type EphemeralTester struct {
	MihomoBin string
	Store     *store.Store
	Log       *slog.Logger
	Activity  func(context.Context, string)
}

func (e *EphemeralTester) TestProxyDelay(ctx context.Context, proxyName string) (int, error) {
	return 0, fmt.Errorf("use TestProfile with store.Profile")
}

// TestProfile starts ephemeral mihomo for one profile and measures delay.
func (e *EphemeralTester) TestProfile(ctx context.Context, p store.Profile) (int, error) {
	perProxyMs := 8000
	if e.Store != nil {
		if t := e.Store.ConnectionTestTimeoutMs(ctx); t > 0 {
			perProxyMs = t
		}
	}
	budget := pretestSingleAPIWait + time.Duration(perProxyMs)*time.Millisecond + 8*time.Second
	tctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	return e.testProfileWithTimeout(tctx, p, perProxyMs)
}

func (e *EphemeralTester) testProfileWithTimeout(ctx context.Context, p store.Profile, perProxyMs int) (int, error) {
	dir, err := os.MkdirTemp("", "muhomor-pretest-*")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(dir)

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
	yaml, proxyName, err := configgen.BuildFromProfile(p, opt, []string{"MATCH,PROXY"})
	if err != nil {
		return 0, err
	}
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		return 0, err
	}
	client := mihomo.NewClient(mihomo.ClientOptions{
		BinPath:         e.mihomoBinFast(),
		ConfigPath:      cfgPath,
		ConfigDir:       dir,
		Controller:      ctl,
		Secret:          opt.Secret,
		APIReadyTimeout: pretestSingleAPIWait,
	})
	if err := client.Start(ctx); err != nil {
		return 0, err
	}
	defer client.Stop()

	testURL := ""
	if e.Store != nil {
		testURL = e.Store.ConnectionTestURL(ctx)
	}
	return client.ProxyDelay(ctx, proxyName, testURL, perProxyMs)
}
