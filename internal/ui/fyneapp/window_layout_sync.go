//go:build cgo

package fyneapp

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func syncWindowLayout(w fyne.Window, inner *container.InnerWindow) {
	if w == nil {
		return
	}
	fyne.Do(func() {
		size := syncNativeWindowSize(w)
		if size.Width <= 0 || size.Height <= 0 {
			return
		}
		if inner != nil {
			inner.Resize(size)
			inner.Refresh()
		}
		if content := w.Content(); content != nil {
			content.Refresh()
			w.Canvas().Refresh(content)
		}
		applyWindowShape(w)
	})
}

func wireWindowLayoutSync(w fyne.Window, inner *container.InnerWindow) {
	if w == nil || inner == nil {
		return
	}
	sync := func() { syncWindowLayout(w, inner) }
	prev := inner.OnResized
	inner.OnResized = func(ev *fyne.DragEvent) {
		if prev != nil {
			prev(ev)
		}
		resizeWindowBy(w, ev.Dragged)
		sync()
	}
}
