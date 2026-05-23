package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/paths"
)

// EnsureDaemon starts muhomor --daemon if API is unreachable.
func EnsureDaemon(ctx context.Context, layout paths.Layout, extraArgs []string) error {
	api := &apiclient.Client{Dial: apiclient.DialConfig{SocketPath: layout.SocketPath()}}
	if api.Reachable(ctx) == nil {
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// If we're muhomor-gui, sibling muhomor.exe should exist.
	muExe := filepath.Join(filepath.Dir(exe), "muhomor.exe")
	if _, err := os.Stat(muExe); err != nil {
		muExe = filepath.Join(filepath.Dir(exe), "muhomor")
		if _, err2 := os.Stat(muExe); err2 != nil {
			muExe = "muhomor"
		}
	}
	args := append([]string{"-dir", layout.DataDir, "--daemon"}, extraArgs...)
	cmd := exec.CommandContext(ctx, muExe, args...)
	AttachDaemonLog(cmd, layout.CacheDir)
	if err := startDaemonProcess(cmd); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if api.Reachable(ctx) == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}
	}
	return fmt.Errorf("daemon did not become reachable")
}

// StopDaemon requests daemon shutdown via HTTP and waits until API is down.
func StopDaemon(ctx context.Context, layout paths.Layout) error {
	api := &apiclient.Client{Dial: apiclient.DialConfig{SocketPath: layout.SocketPath()}}
	if api.Reachable(ctx) != nil {
		return fmt.Errorf("daemon not running")
	}
	if err := api.ShutdownDaemon(ctx); err != nil {
		return err
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if api.Reachable(ctx) != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
	return fmt.Errorf("daemon still reachable after shutdown")
}
