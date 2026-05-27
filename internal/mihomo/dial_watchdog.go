package mihomo

import (
	"context"
	"strings"
	"time"
)

const (
	dialWatchPollInterval = 2 * time.Second
	dialWatchWindow       = 2 * time.Minute
	dialWatchBurst        = 10
	dialWatchCooldown     = 5 * time.Minute
)

// IsDialTimeoutLine reports mihomo subprocess log lines indicating proxy dial timeouts.
func IsDialTimeoutLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "dial") && strings.Contains(lower, "timeout")
}

// RunDialWatchdog tails the subprocess log while connected and calls onBurst when dial timeouts spike.
func RunDialWatchdog(ctx context.Context, logPath string, startOffset int64, onBurst func()) {
	if onBurst == nil || logPath == "" {
		return
	}
	offset := startOffset
	var hits []time.Time
	var lastBurst time.Time
	ticker := time.NewTicker(dialWatchPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lines, newOffset, err := tailRead(logPath, offset)
			if err != nil {
				continue
			}
			offset = newOffset
			now := time.Now()
			cutoff := now.Add(-dialWatchWindow)
			kept := hits[:0]
			for _, t := range hits {
				if t.After(cutoff) {
					kept = append(kept, t)
				}
			}
			hits = kept
			for _, line := range lines {
				if IsDialTimeoutLine(line) {
					hits = append(hits, now)
				}
			}
			if len(hits) >= dialWatchBurst && now.Sub(lastBurst) >= dialWatchCooldown {
				lastBurst = now
				onBurst()
				hits = nil
			}
		}
	}
}
