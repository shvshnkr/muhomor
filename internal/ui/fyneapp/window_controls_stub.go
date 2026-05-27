//go:build !windows && cgo

package fyneapp

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func wireWindowControls(_ fyne.Window, _ *container.InnerWindow) {}

func nativeHideWindow(w fyne.Window) { hideToTray(w) }
