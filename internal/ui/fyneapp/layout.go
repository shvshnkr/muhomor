//go:build cgo

package fyneapp

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	windowSimpleW = 440
	windowSimpleH = 440
	windowExtW    = 960
	windowExtH    = 720
	listMinW      = 420
	listMinH      = 360
	sidebarW      = 280
)

func sectionHeader(title, subtitle string) fyne.CanvasObject {
	titleLbl := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	if subtitle == "" {
		return titleLbl
	}
	sub := widget.NewLabel(subtitle)
	sub.Importance = widget.LowImportance
	sub.Wrapping = fyne.TextWrapWord
	return container.NewVBox(titleLbl, sub)
}

// hintLabel is secondary help text (tooltips are not reliable on all widgets).
func hintLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Importance = widget.LowImportance
	l.Wrapping = fyne.TextWrapWord
	return l
}

func applyTooltip(obj fyne.CanvasObject, text string) {
	if text == "" {
		return
	}
	type tipSetter interface{ SetToolTip(string) }
	if t, ok := obj.(tipSetter); ok {
		t.SetToolTip(text)
	}
}

func formField(label string, field fyne.CanvasObject, tooltip string) fyne.CanvasObject {
	lbl := widget.NewLabel(label)
	applyTooltip(lbl, tooltip)
	applyTooltip(field, tooltip)
	if tooltip == "" {
		return container.NewVBox(lbl, field)
	}
	return container.NewVBox(lbl, field, hintLabel(tooltip))
}

func surfaceCard(inner fyne.CanvasObject, pad float32) fyne.CanvasObject {
	bg := canvas.NewRectangle(colorSurface)
	bg.CornerRadius = 10
	if pad <= 0 {
		pad = themePadding()
	}
	return container.NewStack(bg, container.NewPadded(inner))
}

func accentCard(inner fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(colorSurface2)
	bg.CornerRadius = 10
	bg.StrokeColor = color.NRGBA{R: 0x2d, G: 0xd4, B: 0xbf, A: 0x30}
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.NewPadded(inner))
}

func vSpacer(h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(1, h))
	return r
}

func backButton(label string, fn func()) *widget.Button {
	b := widget.NewButton(label, fn)
	b.Importance = widget.LowImportance
	return b
}

func scrollContent(inner fyne.CanvasObject) fyne.CanvasObject {
	s := container.NewScroll(inner)
	s.SetMinSize(fyne.NewSize(listMinW, listMinH))
	return s
}

// listPanel wraps a list/scroll area so Border layouts cannot collapse it to ~2 lines.
// listPanel gives list widgets a minimum viewport (Border/VBox otherwise squashes to ~2 rows).
func listPanel(inner fyne.CanvasObject) fyne.CanvasObject {
	s := container.NewScroll(inner)
	s.SetMinSize(fyne.NewSize(listMinW, listMinH))
	return surfaceCard(s, themePadding())
}

func themePadding() float32 {
	return 10
}

func statusDot(fill color.Color) *canvas.Circle {
	c := canvas.NewCircle(fill)
	c.Resize(fyne.NewSize(12, 12))
	return c
}

func centeredStatusBlock(dot *canvas.Circle, main, sub *widget.Label) fyne.CanvasObject {
	return container.NewVBox(
		container.NewCenter(dot),
		vSpacer(6),
		container.NewCenter(main),
		container.NewCenter(sub),
	)
}

func windowRoot(inner fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(colorBG)
	return container.NewStack(bg, container.NewPadded(inner))
}
