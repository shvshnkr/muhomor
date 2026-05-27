//go:build !windows

package platform

// ShowErrorMessage is a no-op on non-Windows platforms.
func ShowErrorMessage(title, message string) {
	_, _ = title, message
}
