package apiclient

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDialContext_unixWhenSocketExists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket dial test is for non-windows builds")
	}
	dir := t.TempDir()
	sock := filepath.Join(dir, "muhomor.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Skip("unix listen:", err)
	}
	defer ln.Close()
	go func() {
		c, _ := ln.Accept()
		if c != nil {
			_, _ = c.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
			c.Close()
		}
	}()
	d := DialConfig{SocketPath: sock}
	conn, err := d.DialContext(context.Background(), "tcp", "ignored:8751")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := os.Stat(sock); err != nil {
		t.Fatal(err)
	}
}
