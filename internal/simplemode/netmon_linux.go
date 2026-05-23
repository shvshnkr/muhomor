//go:build linux

package simplemode

import (
	"context"
	"net"
	"sync"
	"time"
)

// pollFallback watches interfaces every 2s when rtnetlink unavailable.
func (n *NetworkMonitor) pollFallback(ctx context.Context) {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	var mu sync.Mutex
	last := ""
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			iface := pollDefaultInterface()
			mu.Lock()
			changed := last != "" && iface != "" && last != iface
			prev := last
			if iface != "" {
				last = iface
			}
			mu.Unlock()
			if changed {
				if n.Log != nil {
					n.Log.Info("network handoff", "from", prev, "to", iface, "event", "H30")
				}
				n.OnHandoff(ctx, "network_handoff")
			}
		}
	}
}

func pollDefaultInterface() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		if len(addrs) > 0 {
			return iface.Name
		}
	}
	return ""
}
