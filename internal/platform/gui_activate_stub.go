//go:build !windows

package platform

// ActivateGUIWindow is a no-op on non-Windows platforms.
func ActivateGUIWindow(title string) bool {
	_ = title
	return false
}
