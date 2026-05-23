//go:build !linux

package selector

import "github.com/muhomor/muhomor/internal/mihomo"

func (e *EphemeralTester) mihomoBinFast() string {
	if e.MihomoBin != "" {
		return e.MihomoBin
	}
	return mihomo.ResolveBin()
}
