//go:build windows

package console

import "golang.org/x/sys/windows"

// PrepareConsole enables ANSI colors on Windows 10+ terminals when possible.
func PrepareConsole() {
	stdout := windows.Handle(windows.Stdout)
	var mode uint32
	if err := windows.GetConsoleMode(stdout, &mode); err != nil {
		return
	}
	_ = windows.SetConsoleMode(stdout, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
