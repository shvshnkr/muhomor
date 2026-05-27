//go:build cgo

package fyneapp

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const (
	windowSimpleW = 480
	windowSimpleH = 460
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
	sub := captionLabel(subtitle)
	sub.Wrapping = fyne.TextWrapWord
	return container.NewVBox(titleLbl, sub)
}

func hintLabel(text string) *widget.Label {
	return captionLabel(text)
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
	return elevatedCard(1, inner, pad)
}

func elevatedCard(level int, inner fyne.CanvasObject, pad float32) fyne.CanvasObject {
	var fill color.Color
	switch level {
	case 2:
		fill = colorSurface2
	default:
		fill = colorSurface
	}
	bg := canvas.NewRectangle(fill)
	bg.CornerRadius = radiusLg
	bg.StrokeColor = colorBorder
	bg.StrokeWidth = 1
	if pad <= 0 {
		pad = cardPadding()
	}
	return container.NewStack(bg, container.NewPadded(inner))
}

func accentCard(inner fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(colorSurface)
	bg.CornerRadius = radiusLg
	bg.StrokeColor = accentStroke()
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.NewPadded(inner))
}

func vSpace(h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(1, h))
	return r
}

func vSpacer(h float32) fyne.CanvasObject { return vSpace(h) }

func backButton(label string, fn func()) *widget.Button {
	return secondaryButton(label, fn)
}

func scrollContent(inner fyne.CanvasObject) fyne.CanvasObject {
	s := container.NewScroll(inner)
	s.SetMinSize(fyne.NewSize(listMinW, listMinH))
	return s
}

func listPanel(inner fyne.CanvasObject) fyne.CanvasObject {
	s := container.NewScroll(inner)
	s.SetMinSize(fyne.NewSize(listMinW, listMinH))
	return surfaceCard(s, cardPadding())
}

func themePadding() float32 { return space3 }

func statusDot(fill color.Color) *canvas.Circle {
	c := canvas.NewCircle(fill)
	c.Resize(fyne.NewSize(statusDotSize, statusDotSize))
	return c
}

func statusDotWithRing(fill color.Color, ring bool) *canvas.Circle {
	d := statusDot(fill)
	if ring {
		d.StrokeColor = accentStroke()
		d.StrokeWidth = 2
	}
	return d
}

// statusHeroRow — dot left, display title + caption activity right.
func statusHeroRow(dot *canvas.Circle, main, sub *widget.Label) fyne.CanvasObject {
	dotCell := container.NewGridWrap(fyne.NewSize(statusDotSize+6, statusDotSize+6), container.NewCenter(dot))
	texts := container.NewVBox(main, sub)
	return container.NewHBox(dotCell, texts)
}

func subtleDivider() fyne.CanvasObject {
	line := canvas.NewRectangle(colorBorder)
	line.SetMinSize(fyne.NewSize(0, 1))
	return line
}

func simpleBody(top, middle, bottom fyne.CanvasObject) fyne.CanvasObject {
	scroll := container.NewScroll(middle)
	return container.NewBorder(top, bottom, nil, nil, scroll)
}

func simpleBodyCompact(top, middle, bottom fyne.CanvasObject) fyne.CanvasObject {
	// Spacer absorbs extra height so hero cards (Stack) do not stretch in extended tabs.
	return container.NewBorder(top, bottom, nil, nil, container.NewVBox(middle, layout.NewSpacer()))
}

// connectButtonRow — full-width primary action with minimum touch height.
func connectButtonRow(btn *widget.Button) fyne.CanvasObject {
	minH := canvas.NewRectangle(color.Transparent)
	minH.SetMinSize(fyne.NewSize(1, connectBtnMinH))
	return container.NewGridWithColumns(1, container.NewStack(minH, btn))
}

func simplePadded(inner fyne.CanvasObject) fyne.CanvasObject {
	return container.NewPadded(inner)
}
