package probe

import "time"

// Config holds 2K probe scheduler knobs (defaults from docs/2K_TELEMETRY.md).
type Config struct {
	TCPTimeout       time.Duration
	TCPWorkers       int
	TCPBatchPerTick  int
	TickInterval     time.Duration
	SuspectRetry     time.Duration
	WarmMaxAge       time.Duration
	ConnectLiveCap   int
	WarmSpotCap      int // live TCP re-check on warm connect path
	WarmURLCap       int
}

func DefaultConfig() Config {
	return Config{
		TCPTimeout:      2200 * time.Millisecond,
		TCPWorkers:      32,
		TCPBatchPerTick: 64,
		TickInterval:    45 * time.Second,
		SuspectRetry:    60 * time.Second,
		WarmMaxAge:      20 * time.Minute,
		ConnectLiveCap:  48,
		WarmSpotCap:     24,
		WarmURLCap:      16,
	}
}
