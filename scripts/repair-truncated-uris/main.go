// One-off: repair profiles stored as "://…" (stripped scheme). Usage:
//   go run ./scripts/repair-truncated-uris -db path/to/muhomor.db
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

func main() {
	db := flag.String("db", "", "path to muhomor.db")
	flag.Parse()
	if *db == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/repair-truncated-uris -db <muhomor.db>")
		os.Exit(2)
	}
	st, err := store.Open(*db)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer st.Close()
	n, err := subscription.RepairTruncatedURIs(context.Background(), st)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("repaired %d profile URIs in %s\n", n, *db)
}
