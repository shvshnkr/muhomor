package apiclient

import (
	"context"
	"net"
	"os"
	"runtime"

	"github.com/muhomor/muhomor/internal/paths"
)

// DefaultTCP is used when Unix socket is unavailable (Windows).
func DefaultTCP() string { return paths.DefaultDaemonTCP() }

// DialConfig resolves how to reach the daemon HTTP API.
type DialConfig struct {
	SocketPath string
	TCPAddr    string
}

func (d DialConfig) tcpAddr() string {
	if d.TCPAddr != "" {
		return d.TCPAddr
	}
	return DefaultTCP()
}

func (d DialConfig) transport() *net.Dialer {
	return &net.Dialer{}
}

// DialContext connects via Unix socket on Linux/macOS if present, else TCP (always on Windows).
func (d DialConfig) DialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	if runtime.GOOS != "windows" && d.SocketPath != "" {
		if _, err := os.Stat(d.SocketPath); err == nil {
			return d.transport().DialContext(ctx, "unix", d.SocketPath)
		}
	}
	return d.transport().DialContext(ctx, "tcp", d.tcpAddr())
}
