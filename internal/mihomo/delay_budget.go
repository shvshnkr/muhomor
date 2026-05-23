package mihomo

// GroupDelayAPITimeout computes the ?timeout= query for GET /group/.../delay.
// In mihomo this value caps the whole parallel URLTest, not a single proxy.
func GroupDelayAPITimeout(perProxyMs, proxyCount int) int {
	if perProxyMs <= 0 {
		perProxyMs = 8000
	}
	if proxyCount <= 0 {
		proxyCount = 1
	}
	// Parallel URLTest shares one context; allow per-proxy budget + headroom for slow nodes.
	total := perProxyMs + (proxyCount*perProxyMs)/6
	if total < 30000 {
		return 30000
	}
	if total > 120000 {
		return 120000
	}
	return total
}
