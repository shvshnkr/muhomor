//go:build cgo

package fyneapp

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type framedWindow struct {
	win   fyne.Window
	inner *container.InnerWindow
}

func newFramedWindow(a fyne.App, title string) framedWindow {
	icon := appIcon()
	if drv, ok := a.Driver().(desktop.Driver); ok {
		w := drv.CreateSplashWindow()
		w.SetPadded(false)
		w.SetTitle(title)
		w.SetIcon(icon)
		inner := container.NewInnerWindow(title, canvas.NewRectangle(colorBG))
		inner.Icon = icon
		inner.Alignment = widget.ButtonAlignTrailing
		wireWindowDrag(w, inner)
		wireWindowControls(w, inner)
		wireWindowLayoutSync(w, inner)
		w.SetContent(inner)
		return framedWindow{win: w, inner: inner}
	}
	w := a.NewWindow(title)
	w.SetIcon(icon)
	inner := container.NewInnerWindow(title, canvas.NewRectangle(colorBG))
	inner.Icon = icon
	wireWindowControls(w, inner)
	wireWindowLayoutSync(w, inner)
	w.SetContent(inner)
	return framedWindow{win: w, inner: inner}
}

func (f framedWindow) setBody(content fyne.CanvasObject) {
	f.inner.SetContent(content)
}

func (f framedWindow) setTitle(title string) {
	f.inner.SetTitle(title)
}

func (f framedWindow) setCloseIntercept(fn func()) {
	f.inner.CloseIntercept = fn
}
