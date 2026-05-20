package apiclient

import (
	"context"
	"net"
	"os"
)

// DefaultTCP is used when Unix socket is unavailable (Windows).
const DefaultTCP = "127.0.0.1:8751"

// DialConfig resolves how to reach the daemon HTTP API.
type DialConfig struct {
	SocketPath string
	TCPAddr    string
}

func (d DialConfig) tcpAddr() string {
	if d.TCPAddr != "" {
		return d.TCPAddr
	}
	return DefaultTCP
}

func (d DialConfig) transport() *net.Dialer {
	return &net.Dialer{}
}

// DialContext connects via Unix socket if present, else TCP.
func (d DialConfig) DialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	if d.SocketPath != "" {
		if _, err := os.Stat(d.SocketPath); err == nil {
			return d.transport().DialContext(ctx, "unix", d.SocketPath)
		}
	}
	return d.transport().DialContext(ctx, "tcp", d.tcpAddr())
}
