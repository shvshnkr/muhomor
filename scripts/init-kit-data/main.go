// Init-kit-data creates a fresh portable kit data/ tree (settings only, no profiles/logs).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/store"
)

func main() {
	out := flag.String("o", "", "output data directory (e.g. dist/muhomor-kit/data)")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/init-kit-data -o <data-dir>")
		os.Exit(2)
	}
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("kit data initialized:", *out)
}

func run(out string) error {
	for _, sub := range []string{"cache", filepath.Join("run", "mihomo")} {
		if err := os.MkdirAll(filepath.Join(out, sub), 0o700); err != nil {
			return err
		}
	}
	for _, name := range []string{
		"cache/activity.log",
		"cache/daemon-debug.err.log",
		"cache/daemon-debug.log",
		"cache/simple-mode.log",
		"cache/desktop-control-status.txt",
		"cache/desktop-control-ping.txt",
		"cache/desktop-control-export.txt",
		"run/mihomo/mihomo-subprocess.log",
	} {
		p := filepath.Join(out, name)
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			return err
		}
	}
	dbPath := filepath.Join(out, "muhomor.db")
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, wal := range []string{dbPath + "-wal", dbPath + "-shm"} {
		_ = os.Remove(wal)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}

	ctx := context.Background()
	set := store.DefaultSettings()
	set.MixedPort = paths.PortableMixedPort // portable kit default; desktop stays 7890
	if err := st.SaveSettings(ctx, set); err != nil {
		return err
	}
	if err := st.Close(); err != nil {
		return err
	}
	st = nil
	for _, wal := range []string{dbPath + "-wal", dbPath + "-shm"} {
		_ = os.Remove(wal)
	}
	return nil
}
