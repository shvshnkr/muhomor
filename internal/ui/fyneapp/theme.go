//go:build cgo

package fyneapp

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
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
	case theme.ColorNameHeaderBackground:
		return colorSurface
	case theme.ColorNameMenuBackground:
		return colorSurface
	case theme.ColorNameInputBackground:
		return colorSurface2
	case theme.ColorNameButton:
		return colorSurface2
	case theme.ColorNamePrimary:
		return colorPrimary
	case theme.ColorNameForeground:
		return colorFG
	case theme.ColorNameInputBorder:
		return colorBorderStrong
	case theme.ColorNamePlaceHolder:
		return colorTertiary
	case theme.ColorNameDisabled:
		return colorTertiary
	case theme.ColorNameHover:
		return colorSurface3
	case theme.ColorNamePressed:
		return colorSurface3
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
		return accentMutedFill()
	case theme.ColorNameOverlayBackground:
		return colorSurface
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
		return space3
	case theme.SizeNameInnerPadding:
		return space2
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameInputBorder:
		return 1
	case theme.SizeNameScrollBar:
		return 6
	case theme.SizeNameScrollBarSmall:
		return 4
	case theme.SizeNameWindowTitleBarHeight:
		return 32
	case theme.SizeNameText:
		return fontBody
	case theme.SizeNameHeadingText:
		return fontDisplay
	case theme.SizeNameSubHeadingText:
		return fontTitle
	case theme.SizeNameCaptionText:
		return fontCaption
	default:
		return t.base.Size(name)
	}
}
