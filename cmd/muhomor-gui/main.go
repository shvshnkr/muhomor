//go:build cgo

// muhomor-gui — Fyne desktop UI (Phase 4.1).
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/ui/fyneapp"
)

func main() {
	dataDir := flag.String("dir", "", "data directory")
	flag.StringVar(dataDir, "d", "", "data directory")
	serviceMode := flag.String("service-mode", "proxy", "proxy|vpn")
	mixedPort := flag.Int("mixed-port", 7890, "mixed inbound port for daemon")
	flag.Parse()

	if *dataDir == "" {
		*dataDir = os.Getenv("MUHOMOR_DATA_DIR")
	}
	layout := paths.Default(*dataDir)
	_ = layout.Ensure()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	opt := fyneapp.Options{
		Layout:      layout,
		ServiceMode: *serviceMode,
		MixedPort:   *mixedPort,
		DaemonArgs: []string{
			"--service-mode", *serviceMode,
			"--mixed-port", strconv.Itoa(*mixedPort),
		},
	}

	if err := fyneapp.Run(ctx, opt); err != nil {
		os.Exit(1)
	}
}
