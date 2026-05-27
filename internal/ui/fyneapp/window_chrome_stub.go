//go:build !windows && cgo

package fyneapp

import "fyne.io/fyne/v2"

func applyWindowShape(_ fyne.Window) {}
