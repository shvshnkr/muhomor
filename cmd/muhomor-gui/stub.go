//go:build fyne && !cgo

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Fyne GUI requires CGO and gcc (legacy -tags fyne build).")
	fmt.Fprintln(os.Stderr, "Default GUI is Wails: wails build -platform windows/amd64 -o muhomor-gui.exe")
	os.Exit(1)
}
