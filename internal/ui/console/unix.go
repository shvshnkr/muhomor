//go:build !windows

package console

// PrepareConsole is a no-op on Unix (ANSI works in most terminals).
func PrepareConsole() {}
