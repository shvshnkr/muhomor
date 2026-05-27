package wailsapp

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"sync"
)

var (
	iconOnce sync.Once
	iconRGBA *image.RGBA
	iconPNG  []byte
)

// TrayIconPNG returns a 64×64 PNG (Linux/macOS systray).
func TrayIconPNG() []byte {
	iconOnce.Do(initIconCache)
	return iconPNG
}

func initIconCache() {
	iconRGBA = renderIconRGBA()
	var buf bytes.Buffer
	_ = png.Encode(&buf, iconRGBA)
	iconPNG = buf.Bytes()
}

// renderIconRGBA draws the app mark; tuned for 16px tray/taskbar.
func renderIconRGBA() *image.RGBA {
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
	return img
}
