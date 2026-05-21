package probe

import (
	"context"
	"time"

	"github.com/muhomor/muhomor/internal/store"
)

const (
	PresetLow    = "low"
	PresetNormal = "normal"
	PresetHigh   = "high"
)

// ConfigForPreset returns scheduler/connect knobs for host capacity.
func ConfigForPreset(preset string) Config {
	switch preset {
	case PresetLow:
		return Config{
			TCPTimeout:      2500 * time.Millisecond,
			TCPWorkers:      16,
			TCPBatchPerTick: 32,
			TickInterval:    60 * time.Second,
			SuspectRetry:    90 * time.Second,
			WarmMaxAge:      15 * time.Minute,
			ConnectLiveCap:  64,
			WarmSpotCap:     16,
			WarmURLCap:      12,
		}
	case PresetHigh:
		return Config{
			TCPTimeout:      2000 * time.Millisecond,
			TCPWorkers:      48,
			TCPBatchPerTick: 96,
			TickInterval:    30 * time.Second,
			SuspectRetry:    45 * time.Second,
			WarmMaxAge:      25 * time.Minute,
			ConnectLiveCap:  128,
			WarmSpotCap:     32,
			WarmURLCap:      20,
		}
	default:
		c := DefaultConfig()
		c.WarmSpotCap = 24
		c.WarmURLCap = 16
		return c
	}
}

// ConfigFromStore reads probe_preset KV (default normal).
func ConfigFromStore(ctx context.Context, st *store.Store) Config {
	if st == nil {
		return DefaultConfig()
	}
	return ConfigForPreset(st.ProbePreset(ctx))
}
