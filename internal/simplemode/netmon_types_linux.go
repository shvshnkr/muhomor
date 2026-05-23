//go:build linux

package simplemode

import (
	"context"
	"log/slog"
)

// NetworkMonitor watches default route / interfaces for handoff (adapt).
type NetworkMonitor struct {
	Log       *slog.Logger
	OnHandoff func(ctx context.Context, reason string)
}

func (n *NetworkMonitor) Stop() {}
