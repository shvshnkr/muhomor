// muhomor — Dahusim logic on mihomo (Linux desktop).
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/controller"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
	"github.com/muhomor/muhomor/internal/systemd"
	"github.com/muhomor/muhomor/internal/ui/pseudogui"
)

func main() {
	dataDir := flag.String("dir", "", "data directory")
	flag.StringVar(dataDir, "d", "", "data directory (alias)")
	daemon := flag.Bool("daemon", false, "run headless daemon")
	ctl := flag.String("ctl", "", "control: start|stop|reload|status|ping|export-log|update-check|chain")
	systemdCmd := flag.String("systemd", "", "systemd: install|...")
	systemdScope := flag.String("systemd-scope", "user", "systemd scope")
	importURI := flag.String("import-uri", "", "import proxy URI")
	importFile := flag.String("import-file", "", "import subscription file")
	genYAML := flag.String("gen-yaml", "", "print mihomo yaml for uri")
	listProfiles := flag.Bool("profiles", false, "list stored profiles")
	chainIDs := flag.String("chain", "", "profile ids for chain (with --ctl chain)")
	serviceMode := flag.String("service-mode", "", "proxy|vpn")
	mixedPort := flag.Int("mixed-port", 0, "mixed proxy port")
	proxyAuth := flag.String("proxy-auth", "", "user:password or 'none'")
	routeQuick := flag.Int("route-quick-profile", -1, "0=manual 1=ru_direct 2=ru_blocked_ai")
	pseudoGUI := flag.Bool("pseudo-gui", false, "interactive terminal pseudo-GUI")
	flag.Parse()

	if *dataDir == "" {
		*dataDir = os.Getenv("MUHOMOR_DATA_DIR")
	}
	layout := paths.Default(*dataDir)
	_ = layout.Ensure()

	if *listProfiles {
		runListProfiles(layout)
		return
	}
	if *genYAML != "" {
		runGenYAML(*genYAML)
		return
	}
	if *importURI != "" || *importFile != "" {
		if err := runImport(layout, *importURI, *importFile); err != nil {
			fmt.Fprintf(os.Stderr, "import: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *systemdCmd != "" {
		exe, _ := os.Executable()
		launcher := buildDaemonLauncher(exe, layout, *serviceMode, *mixedPort, *proxyAuth, *routeQuick)
		if err := systemd.Run(*systemdCmd, *systemdScope, launcher); err != nil {
			fmt.Fprintf(os.Stderr, "systemd: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *pseudoGUI {
		os.Exit(pseudogui.Start(context.Background(), layout, pseudogui.Options{
			DaemonArgs: pseudogui.DaemonArgsFromCLI(*serviceMode, *mixedPort, *proxyAuth, *routeQuick),
		}))
	}
	if *ctl == "chain" {
		if *chainIDs == "" {
			fmt.Fprintln(os.Stderr, "use: muhomor --ctl chain --chain 1,2,3")
			os.Exit(2)
		}
		if err := controller.RunCtlChain(layout, *chainIDs); err != nil {
			fmt.Fprintf(os.Stderr, "ctl: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *ctl != "" {
		if err := controller.RunCtl(layout, *ctl); err != nil {
			fmt.Fprintf(os.Stderr, "ctl: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if !*daemon {
		printUsage()
		os.Exit(2)
	}
	runDaemon(layout, *serviceMode, *mixedPort, *proxyAuth, *routeQuick)
}

func buildDaemonLauncher(exe string, layout paths.Layout, serviceMode string, mixedPort int, proxyAuth string, routeQuick int) []string {
	args := []string{exe, "-dir", layout.DataDir, "--daemon"}
	if serviceMode != "" {
		args = append(args, "--service-mode", serviceMode)
	}
	if mixedPort > 0 {
		args = append(args, "--mixed-port", strconv.Itoa(mixedPort))
	}
	if proxyAuth != "" {
		args = append(args, "--proxy-auth", proxyAuth)
	}
	if routeQuick >= 0 {
		args = append(args, "--route-quick-profile", strconv.Itoa(routeQuick))
	}
	return args
}

func runDaemon(layout paths.Layout, serviceMode string, mixedPort int, proxyAuth string, routeQuick int) {
	st, err := store.Open(layout.DBPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()
	ctx := context.Background()
	_ = controller.ApplyCLISettings(ctx, st, serviceMode, mixedPort, proxyAuth, routeQuick, serviceMode == "vpn")

	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	rt := controller.NewRuntime(layout, st, log)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	d := &controller.Daemon{Runtime: rt, Log: log, ShutdownFn: cancel}
	rt.SetDaemonContext(ctx)

	go func() {
		<-ctx.Done()
		_ = rt.Stop(context.Background())
	}()

	if err := d.ListenAndServe(ctx, layout.SocketPath()); err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
		os.Exit(1)
	}
}

func runListProfiles(layout paths.Layout) {
	st, err := store.Open(layout.DBPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()
	list, err := st.ListAllProfiles(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "list: %v\n", err)
		os.Exit(1)
	}
	for _, p := range list {
		fmt.Printf("id=%d type=%s name=%q enabled=%t wl_builtin=%t\n", p.ID, p.Type, p.Name, p.Enabled, p.WLBuiltinPool)
	}
}

func runImport(layout paths.Layout, uri, file string) error {
	st, err := store.Open(layout.DBPath())
	if err != nil {
		return err
	}
	defer st.Close()
	ctx := context.Background()
	var lines []string
	if uri != "" {
		lines = []string{uri}
	} else {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		lines, err = subscription.ParseLines(f)
		if err != nil {
			return err
		}
	}
	for i, line := range lines {
		if reason := subscription.UnsupportedReason(line); reason != "" {
			fmt.Fprintf(os.Stderr, "skip: %s (%s)\n", truncate(line, 60), reason)
			continue
		}
		typ := subscription.Scheme(line)
		name := fmt.Sprintf("imported-%d", i+1)
		if typ == "vless" {
			if p, err := configgen.ParseVLESSURI(line); err == nil && p.Name != "" {
				name = p.Name
			}
		}
		id, err := st.UpsertProfile(ctx, name, typ, line)
		if err != nil {
			return err
		}
		fmt.Printf("profile id=%d name=%s type=%s\n", id, name, typ)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func runGenYAML(uri string) {
	p, err := configgen.ParseVLESSURI(uri)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}
	yaml, _, err := configgen.BuildFromVLESS(p, configgen.DefaultBuildOptions())
	if err != nil {
		fmt.Fprintf(os.Stderr, "build: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(yaml)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `muhomor — Go core for Dahusim on mihomo

  muhomor --daemon [-d PATH] [--service-mode proxy|vpn] [--mixed-port N] [--proxy-auth user:pass|none]
  muhomor --ctl start|stop|reload|status|chain ...
  muhomor --ctl chain --chain 1,2,3
  muhomor --profiles
  muhomor --import-uri URI | --import-file PATH
  muhomor --route-quick-profile 0|1|2
  muhomor --pseudo-gui [-d PATH]
  muhomor --systemd install|...

`)
}
