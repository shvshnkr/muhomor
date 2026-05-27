//go:build cgo

package fyneapp

import "fyne.io/fyne/v2"

func quitApp(a fyne.App) {
	if a == nil {
		return
	}
	fyne.Do(func() {
		a.Quit()
	})
}

func hideToTray(w fyne.Window) {
	if w == nil {
		return
	}
	fyne.Do(func() {
		w.Hide()
	})
}
