package wailsapp

import (
	"bytes"
	"testing"
)

func TestEncodeICO(t *testing.T) {
	b := encodeICO(renderIconRGBA(), []int{16, 32})
	if len(b) < 22 {
		t.Fatalf("ico too short: %d", len(b))
	}
	if b[0] != 0 || b[1] != 0 || b[2] != 1 || b[3] != 0 {
		t.Fatalf("bad ico header: %v", b[:4])
	}
	if b[4] != 2 || b[5] != 0 {
		t.Fatalf("expected 2 images, got count %d", uint16(b[4])|uint16(b[5])<<8)
	}
	if !bytes.Contains(b, []byte{0x89, 0x50, 0x4e, 0x47}) {
		t.Fatal("ico should embed PNG")
	}
}
