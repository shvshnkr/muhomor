package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/muhomor/muhomor/internal/store"
)

func main() {
	db := flag.String("db", "", "path to muhomor.db")
	flag.Parse()
	if *db == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/verify-kit-settings -db <path>")
		os.Exit(2)
	}
	st, err := store.Open(*db)
	if err != nil {
		panic(err)
	}
	defer st.Close()
	set, err := st.LoadSettings(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Printf("service_mode=%s mixed_port=%d aggregation=%s bulk=%v multipath=%v\n",
		set.ServiceMode, set.MixedPort, set.AggregationMode, set.BulkEnabled, set.MultipathEnabled)
}
