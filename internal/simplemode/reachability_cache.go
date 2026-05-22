package simplemode

import (
	"sync"
	"time"

	"github.com/muhomor/muhomor/internal/reachability"
)

// ReachabilityCache grace cache during tunnel restart (SimpleModeTunnelRestart subset).
type ReachabilityCache struct {
	mu        sync.RWMutex
	cached    reachability.Result
	expiresAt time.Time
}

func (c *ReachabilityCache) Put(r reachability.Result, grace time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cached = r
	c.expiresAt = time.Now().Add(grace)
}

func (c *ReachabilityCache) Get() (reachability.Result, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Now().After(c.expiresAt) {
		return reachability.Result{}, false
	}
	return c.cached, true
}

// Invalidate drops cached reachability (network handoff, fresh connect probe).
func (c *ReachabilityCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cached = reachability.Result{}
	c.expiresAt = time.Time{}
}
