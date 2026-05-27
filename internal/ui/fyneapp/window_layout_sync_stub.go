//go:build !windows && cgo

package fyneapp

import "fyne.io/fyne/v2"

func syncNativeWindowSize(w fyne.Window) fyne.Size {
	return w.Canvas().Size()
}
