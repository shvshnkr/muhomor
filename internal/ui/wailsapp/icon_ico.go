package wailsapp

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"sync"
)

var (
	icoOnce sync.Once
	iconICO []byte
)

// TrayIconICO returns a multi-size .ico for Windows systray (LoadImage IMAGE_ICON).
func TrayIconICO() []byte {
	icoOnce.Do(func() {
		iconOnce.Do(initIconCache)
		iconICO = encodeICO(iconRGBA, []int{16, 32, 48})
	})
	return iconICO
}

func encodeICO(src *image.RGBA, sizes []int) []byte {
	pngs := make([][]byte, len(sizes))
	for i, sz := range sizes {
		scaled := scaleRGBA(src, sz)
		var buf bytes.Buffer
		_ = png.Encode(&buf, scaled)
		pngs[i] = buf.Bytes()
	}

	var out bytes.Buffer
	_ = binary.Write(&out, binary.LittleEndian, uint16(0)) // reserved
	_ = binary.Write(&out, binary.LittleEndian, uint16(1)) // type: icon
	_ = binary.Write(&out, binary.LittleEndian, uint16(len(sizes)))

	offset := 6 + 16*len(sizes)
	for i, sz := range sizes {
		w, h := byte(sz), byte(sz)
		if sz >= 256 {
			w, h = 0, 0
		}
		entry := make([]byte, 16)
		entry[0] = w
		entry[1] = h
		entry[4] = 1 // color planes
		entry[5] = 0
		entry[6] = 32 // bits per pixel (PNG payload)
		entry[7] = 0
		binary.LittleEndian.PutUint32(entry[8:12], uint32(len(pngs[i])))
		binary.LittleEndian.PutUint32(entry[12:16], uint32(offset))
		_, _ = out.Write(entry)
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		_, _ = out.Write(p)
	}
	return out.Bytes()
}

func scaleRGBA(src *image.RGBA, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			sx := x * sw / size
			sy := y * sh / size
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}
