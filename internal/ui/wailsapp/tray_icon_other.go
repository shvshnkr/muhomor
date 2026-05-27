//go:build !windows

package wailsapp

func trayIconBytes() []byte {
	return TrayIconPNG()
}
