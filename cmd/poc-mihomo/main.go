// poc-mihomo — Phase 0 PoC: VLESS URI → YAML → optional mihomo subprocess.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/mihomo"
)

func main() {
	uri := flag.String("uri", os.Getenv("MUHOMOR_VLESS_URI"), "vless:// profile URI")
	out := flag.String("o", "", "write config.yaml to path (default: stdout only)")
	run := flag.Bool("run", false, "start mihomo with generated config (Linux; needs mihomo in PATH)")
	wait := flag.Duration("wait", 8*time.Second, "how long to keep mihomo running when -run")
	flag.Parse()

	if *uri == "" {
		fmt.Fprintln(os.Stderr, "usage: poc-mihomo -uri 'vless://...' [-o config.yaml] [-run]")
		os.Exit(2)
	}
	prof, err := configgen.ParseVLESSURI(*uri)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}
	opt := configgen.DefaultBuildOptions()
	yaml, proxyName, err := configgen.BuildFromVLESS(prof, opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build: %v\n", err)
		os.Exit(1)
	}
	if *out == "" {
		fmt.Print(yaml)
	} else {
		if err := os.WriteFile(*out, []byte(yaml), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "write: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %s (proxy=%s)\n", *out, proxyName)
	}
	if !*run {
		return
	}
	dir, err := os.MkdirTemp("", "muhomor-poc-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "tmpdir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write config: %v\n", err)
		os.Exit(1)
	}
	client := mihomo.NewClient(mihomo.ClientOptions{
		ConfigPath: cfgPath,
		ConfigDir:  dir,
		Controller: opt.ExternalController,
		Secret:     opt.Secret,
	})
	ctx, cancel := context.WithTimeout(context.Background(), *wait+5*time.Second)
	defer cancel()
	if err := client.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "start mihomo: %v\n", err)
		os.Exit(1)
	}
	defer client.Stop()
	time.Sleep(*wait)
	ver, err := client.Version(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "version API: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "mihomo OK version=%s proxy=%s\n", ver, proxyName)
}
