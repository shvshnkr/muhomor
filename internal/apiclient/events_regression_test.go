package apiclient

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
)

func TestRegression_StreamEvents_wrapsRefusedNotTimeout(t *testing.T) {
	c := &Client{}
	c.initHTTP()
	c.transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, errors.New("connectex: No connection could be made because the target machine actively refused it")
	}

	_, err := c.StreamEvents(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "daemon not running") {
		t.Fatalf("refused should wrap: %v", err)
	}

	c2 := &Client{}
	c2.initHTTP()
	c2.transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, context.DeadlineExceeded
	}
	_, err2 := c2.StreamEvents(context.Background())
	if err2 != nil && strings.Contains(err2.Error(), "daemon not running") {
		t.Fatalf("timeout must not be daemon down: %v", err2)
	}
}
