//go:build !cgo

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "muhomor-gui requires CGO and a C compiler (gcc/MinGW on Windows).")
	fmt.Fprintln(os.Stderr, "Install MinGW-w64, set CGO_ENABLED=1, then: go build -o muhomor-gui.exe ./cmd/muhomor-gui")
	os.Exit(1)
}
