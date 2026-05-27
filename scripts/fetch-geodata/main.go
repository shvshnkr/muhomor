// fetch-geodata downloads geoip.metadb for portable kit packaging.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/muhomor/muhomor/internal/mihomo"
)

func main() {
	out := flag.String("o", "", "output file path (e.g. dist/muhomor-kit/data/run/mihomo/geoip.metadb)")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/fetch-geodata -o <path/to/geoip.metadb>")
		os.Exit(2)
	}
	cfgDir := filepath.Dir(*out)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	if err := mihomo.DownloadGeoDatabase(ctx, cfgDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	st, _ := os.Stat(*out)
	fmt.Printf("geodata: %s (%d bytes)\n", *out, st.Size())
}
