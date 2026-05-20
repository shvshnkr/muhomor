//go:build !linux

package simplemode

import "context"

// NetworkMonitor stub on non-Linux (handoff via manual --ctl adapt).
type NetworkMonitor struct {
	OnHandoff func(ctx context.Context, reason string)
}

func (n *NetworkMonitor) Start(ctx context.Context) {}
func (n *NetworkMonitor) Stop()                     {}
