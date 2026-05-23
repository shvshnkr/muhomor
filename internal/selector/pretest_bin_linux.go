//go:build linux

package selector

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/muhomor/muhomor/internal/mihomo"
)

var (
	fastMihomoBin     string
	fastMihomoBinOnce sync.Once
)

// mihomoBinFast uses a copy under /tmp when the kit lives on NTFS (/mnt/c) so exec and API start stay fast.
func (e *EphemeralTester) mihomoBinFast() string {
	fastMihomoBinOnce.Do(func() {
		src := e.MihomoBin
		if src == "" {
			src = mihomo.ResolveBin()
		}
		if !strings.Contains(src, "/mnt/") {
			fastMihomoBin = src
			return
		}
		dst := filepath.Join(os.TempDir(), "muhomor-mihomo-cached")
		if st, err := os.Stat(src); err == nil {
			if dstSt, err2 := os.Stat(dst); err2 != nil || dstSt.ModTime().Before(st.ModTime()) || dstSt.Size() != st.Size() {
				_ = copyMihomoBin(src, dst)
				_ = os.Chmod(dst, 0o755)
			}
			fastMihomoBin = dst
			return
		}
		fastMihomoBin = src
	})
	return fastMihomoBin
}

func copyMihomoBin(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
