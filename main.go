// muhomor-gui — Wails entry (used by `wails build` / `wails dev` from repo root).
//
// Direct Go build: go build -o muhomor-gui.exe ./cmd/muhomor-gui
package main

import (
	"context"
	"embed"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/platform"
	"github.com/muhomor/muhomor/internal/ui/wailsapp"
)

//go:embed all:cmd/muhomor-gui/frontend/dist
var assets embed.FS

func main() {
	dataDir := flag.String("dir", "", "data directory")
	flag.StringVar(dataDir, "d", "", "data directory")
	serviceMode := flag.String("service-mode", "proxy", "proxy|vpn")
	mixedPort := flag.Int("mixed-port", 0, "mixed inbound port (0=auto: kit 2181 / desktop 7890)")
	routeQuick := flag.Int("route-quick-profile", -1, "0=manual 1=ru_direct 2=ru_blocked_ai 3=wg_over_wl_tunnel")
	startHidden := flag.Bool("start-hidden", false, "start with main window hidden (tray)")
	flag.Parse()

	paths.ApplyPortableKitEnv()
	*mixedPort = paths.DefaultMixedPort(*mixedPort)
	layout := paths.Default(*dataDir)
	_ = layout.Ensure()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	daemonArgs := []string{
		"--service-mode", *serviceMode,
		"--mixed-port", strconv.Itoa(*mixedPort),
	}
	if *routeQuick >= 0 {
		daemonArgs = append(daemonArgs, "--route-quick-profile", strconv.Itoa(*routeQuick))
	}

	platform.StaleGUILock(layout.DataDir)
	guiUnlock, lockErr := platform.AcquireGUILock(layout.DataDir)
	if lockErr != nil {
		if platform.ActivateGUIWindow(wailsapp.WindowTitle()) {
			os.Exit(0)
		}
		platform.ShowErrorMessage(wailsapp.WindowTitle(), lockErr.Error())
		os.Exit(1)
	}

	hidden := wailsapp.StartHiddenEnabled(*startHidden)
	app := wailsapp.NewApp(wailsapp.Options{
		Layout:         layout,
		ServiceMode:    *serviceMode,
		MixedPort:      *mixedPort,
		RouteQuick:     *routeQuick,
		DaemonArgs:     daemonArgs,
		StartHidden:    hidden,
		GUILockRelease: guiUnlock,
	})

	if err := wailsapp.RunDesktop(wailsapp.DesktopRunOptions{App: app, Assets: assets, StartHidden: hidden}); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	_ = ctx
}
