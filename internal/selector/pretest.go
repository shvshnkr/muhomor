package selector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/mihomo"
	"github.com/muhomor/muhomor/internal/store"
)

var pretestPort uint32 = 9000

// EphemeralTester runs url-test via short-lived mihomo (pre-connect, like libcore forTest).
type EphemeralTester struct {
	MihomoBin string
	Store     *store.Store
}

func (e *EphemeralTester) TestProxyDelay(ctx context.Context, proxyName string) (int, error) {
	return 0, fmt.Errorf("use TestProfile with store.Profile")
}

// TestProfile starts ephemeral mihomo for one profile and measures delay.
func (e *EphemeralTester) TestProfile(ctx context.Context, p store.Profile) (int, error) {
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
	ctl := fmt.Sprintf("127.0.0.1:%d", port)
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
	bin := e.MihomoBin
	if bin == "" {
		bin = mihomo.ResolveBin()
	}
	client := mihomo.NewClient(mihomo.ClientOptions{
		BinPath:    bin,
		ConfigPath: cfgPath,
		ConfigDir:  dir,
		Controller: ctl,
		Secret:     opt.Secret,
	})
	if err := client.Start(ctx); err != nil {
		return 0, err
	}
	defer func() {
		client.Stop()
	}()
	testURL := ""
	timeout := 0
	if e.Store != nil {
		testURL = e.Store.ConnectionTestURL(ctx)
		timeout = e.Store.ConnectionTestTimeoutMs(ctx)
	}
	return client.ProxyDelay(ctx, proxyName, testURL, timeout)
}
