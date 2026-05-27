//go:build cgo

package fyneapp

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/muhomor/muhomor/internal/appcore"
)

type routeTab struct {
	content     fyne.CanvasObject
	w           fyne.Window
	app         *appcore.App
	ctx         context.Context
	onBack      func()
	statusLabel *widget.Label
	presets     *segmentedList
}

func newRouteTab(w fyne.Window, app *appcore.App, ctx context.Context, onBack func()) *routeTab {
	r := &routeTab{w: w, app: app, ctx: ctx, onBack: onBack}
	r.statusLabel = captionLabel("")
	r.statusLabel.Wrapping = fyne.TextWrapWord
	opts := []string{
		"0 — Ручной",
		"1 — RU напрямую",
		"2 — RU/заблокированное и AI через прокси",
		"3 — WG over WL tunnel (private через прокси)",
	}
	r.presets = newSegmentedList(opts)
	backBtn := backButton("← Простой режим", func() { r.onBack() })
	applyBtn := widget.NewButton("Применить", func() {
		v := -1
		for i, opt := range r.presets.opts {
			if opt == r.presets.Selected() {
				v = i
				break
			}
		}
		if v < 0 {
			return
		}
		go func() {
			err := r.app.Config.SetRouteQuickProfile(r.ctx, v)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, r.w)
					return
				}
				r.statusLabel.SetText("Маршрут сохранён — применится при connect/reload")
			})
		}()
	})
	applyBtn.Importance = widget.HighImportance

	body := container.NewVBox(
		sectionHeader("Быстрый маршрут", "Пресеты rule-set для .ru и AI (как в Dahusim)."),
		surfaceCard(r.presets.Object(), cardPadding()),
		vSpace(space3),
		applyBtn,
		vSpace(space2),
		r.statusLabel,
	)
	r.content = container.NewPadded(container.NewBorder(backBtn, nil, nil, nil, scrollContent(body)))
	return r
}

func (r *routeTab) load(ctx context.Context) {
	v, err := r.app.Config.RouteQuickProfile(ctx)
	if err != nil {
		return
	}
	fyne.Do(func() {
		if v >= 0 && v < len(r.presets.opts) {
			r.presets.SetSelected(r.presets.opts[v])
		}
	})
}
