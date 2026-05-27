//go:build cgo

package fyneapp

import "image/color"

// Design tokens — dark graphite + teal accent (4 elevation levels).

var (
	colorBG       = color.NRGBA{R: 0x09, G: 0x0b, B: 0x0f, A: 0xff} // bgBase
	colorRaised   = color.NRGBA{R: 0x12, G: 0x15, B: 0x1a, A: 0xff} // bgRaised
	colorSurface  = color.NRGBA{R: 0x18, G: 0x1c, B: 0x22, A: 0xff} // surface1
	colorSurface2 = color.NRGBA{R: 0x1f, G: 0x25, B: 0x2d, A: 0xff} // surface2 / inputs
	colorSurface3 = color.NRGBA{R: 0x28, G: 0x2f, B: 0x38, A: 0xff} // hover / pressed neutral
	colorFG       = color.NRGBA{R: 0xf3, G: 0xf4, B: 0xf6, A: 0xff} // textPrimary
	colorMuted    = color.NRGBA{R: 0x8b, G: 0x94, B: 0x9e, A: 0xff} // textSecondary
	colorTertiary = color.NRGBA{R: 0x5c, G: 0x65, B: 0x70, A: 0xff} // textTertiary
	colorPrimary  = color.NRGBA{R: 0x2d, G: 0xd4, B: 0xbf, A: 0xff} // accent
	colorPrimaryH = color.NRGBA{R: 0x38, G: 0xe0, B: 0xcb, A: 0xff} // accentHover
	colorPrimaryD = color.NRGBA{R: 0x14, G: 0xb8, B: 0xa6, A: 0xff} // accentPressed
	colorSuccess  = color.NRGBA{R: 0x34, G: 0xd3, B: 0x99, A: 0xff}
	colorWarn     = color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff}
	colorError    = color.NRGBA{R: 0xf8, G: 0x71, B: 0x71, A: 0xff}
	colorBorder   = color.NRGBA{R: 0x23, G: 0x29, B: 0x30, A: 0xff} // borderSubtle
	colorBorderStrong = color.NRGBA{R: 0x34, G: 0x3c, B: 0x47, A: 0xff}
)

const (
	space1 = 4
	space2 = 8
	space3 = 12
	space4 = 16
	space5 = 24
	space6 = 32

	radiusSm = 6
	radiusMd = 10
	radiusLg = 12

	fontDisplay  = 28
	fontTitle    = 18
	fontBody     = 14
	fontCaption  = 12
	fontStat     = 22

	connectBtnMinH = 44
	statusDotSize  = 18
)

func accentMutedFill() color.Color {
	return color.NRGBA{R: 0x2d, G: 0xd4, B: 0xbf, A: 0x2e} // ~18%
}

func accentStroke() color.Color {
	return color.NRGBA{R: 0x2d, G: 0xd4, B: 0xbf, A: 0x40}
}
