//go:build cgo

package fyneapp

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"sync"

	"fyne.io/fyne/v2"
)

var (
	iconOnce sync.Once
	iconRes  fyne.Resource
)

func appIcon() fyne.Resource {
	iconOnce.Do(func() {
		iconRes = fyne.NewStaticResource("muhomor.png", renderIconPNG())
	})
	return iconRes
}

func applyAppIcon(a fyne.App, w fyne.Window) {
	icon := appIcon()
	a.SetIcon(icon)
	w.SetIcon(icon)
}

// Simple teal mark on dark background — taskbar / tray readable at 16px.
func renderIconPNG() []byte {
	const n = 64
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	bg := color.NRGBA{R: 0x0f, G: 0x14, B: 0x19, A: 0xff}
	capC := color.NRGBA{R: 0x2d, G: 0xd4, B: 0xbf, A: 0xff}
	stemC := color.NRGBA{R: 0x14, G: 0xb8, B: 0xa6, A: 0xff}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			img.Set(x, y, bg)
		}
	}
	cx, cy, r := float64(n)/2, float64(n)*0.38, float64(n)*0.28
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, capC)
			}
		}
	}
	stemW := float64(n) * 0.16
	stemTop := float64(n) * 0.48
	stemBot := float64(n) * 0.82
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			fx := float64(x) + 0.5
			fy := float64(y) + 0.5
			if fy >= stemTop && fy <= stemBot && fx >= cx-stemW/2 && fx <= cx+stemW/2 {
				img.Set(x, y, stemC)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
