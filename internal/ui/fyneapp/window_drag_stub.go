//go:build !windows && cgo

package fyneapp

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func wireWindowDrag(_ fyne.Window, _ *container.InnerWindow) {}

func resizeWindowBy(_ fyne.Window, _ fyne.Delta) {}
