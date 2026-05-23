//go:build cgo

package fyneapp

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Design palette (dark, teal accent — modern utility app, not terminal dump).
var (
	colorBG       = color.NRGBA{R: 0x0f, G: 0x14, B: 0x19, A: 0xff}
	colorSurface  = color.NRGBA{R: 0x1a, G: 0x22, B: 0x2c, A: 0xff}
	colorSurface2 = color.NRGBA{R: 0x22, G: 0x2d, B: 0x3a, A: 0xff}
	colorPrimary  = color.NRGBA{R: 0x2d, G: 0xd4, B: 0xbf, A: 0xff}
	colorPrimaryD = color.NRGBA{R: 0x14, G: 0xb8, B: 0xa6, A: 0xff}
	colorFG       = color.NRGBA{R: 0xe8, G: 0xec, B: 0xf0, A: 0xff}
	colorMuted    = color.NRGBA{R: 0x94, G: 0xa3, B: 0xb8, A: 0xff}
	colorSuccess  = color.NRGBA{R: 0x34, G: 0xd3, B: 0x99, A: 0xff}
	colorWarn     = color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff}
	colorError    = color.NRGBA{R: 0xf8, G: 0x71, B: 0x71, A: 0xff}
	colorBorder   = color.NRGBA{R: 0x2d, G: 0x3a, B: 0x47, A: 0xff}
)

type muhomorTheme struct {
	base fyne.Theme
}

func newMuhomorTheme() fyne.Theme {
	return &muhomorTheme{base: theme.DefaultTheme()}
}

func applyTheme(a fyne.App) {
	a.Settings().SetTheme(newMuhomorTheme())
}

func (t *muhomorTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colorBG
	case theme.ColorNameInputBackground, theme.ColorNameHeaderBackground, theme.ColorNameMenuBackground:
		return colorSurface
	case theme.ColorNameButton:
		return colorSurface2
	case theme.ColorNamePrimary:
		return colorPrimary
	case theme.ColorNameForeground, theme.ColorNameInputBorder:
		return colorFG
	case theme.ColorNamePlaceHolder, theme.ColorNameDisabled:
		return colorMuted
	case theme.ColorNameHover, theme.ColorNamePressed:
		// Per-importance hover (Danger stays red, High stays teal).
		return t.base.Color(name, variant)
	case theme.ColorNameSeparator, theme.ColorNameScrollBar:
		return colorBorder
	case theme.ColorNameError:
		return colorError
	case theme.ColorNameSuccess:
		return colorSuccess
	case theme.ColorNameWarning:
		return colorWarn
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 0x60}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0x2d, G: 0xd4, B: 0xbf, A: 0x40}
	case theme.ColorNameOverlayBackground:
		return color.NRGBA{R: 0x0a, G: 0x0d, B: 0x10, A: 0xd0}
	default:
		return t.base.Color(name, variant)
	}
}

func (t *muhomorTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *muhomorTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *muhomorTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInnerPadding:
		return 6
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameInputBorder:
		return 1
	case theme.SizeNameScrollBar:
		return 6
	case theme.SizeNameScrollBarSmall:
		return 4
	default:
		return t.base.Size(name)
	}
}
