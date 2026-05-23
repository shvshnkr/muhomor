//go:build linux

package simplemode

import (
	"context"

	"github.com/vishvananda/netlink"
)

// StartRTNetlink uses netlink route updates for handoff (replaces poll when available).
func (n *NetworkMonitor) StartRTNetlink(ctx context.Context) {
	if n.OnHandoff == nil {
		return
	}
	ch := make(chan netlink.RouteUpdate, 8)
	done := make(chan struct{})
	if err := netlink.RouteSubscribe(ch, done); err != nil {
		if n.Log != nil {
			n.Log.Warn("rtnetlink subscribe failed, using poll fallback", "err", err)
		}
		go n.pollFallback(ctx)
		return
	}
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ch:
				if n.Log != nil {
					n.Log.Info("network handoff", "reason", "rtnetlink", "event", "H30")
				}
				n.OnHandoff(ctx, "network_handoff")
			}
		}
	}()
}

// Start uses rtnetlink on Linux (fallback: poll).
func (n *NetworkMonitor) Start(ctx context.Context) {
	n.StartRTNetlink(ctx)
}
