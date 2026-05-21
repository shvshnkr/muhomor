package aggregate

import "time"

// Preset names (KV multipath_preset).
const (
	PresetLow    = "low"
	PresetNormal = "normal"
	PresetHigh   = "high"
)

// Config holds multipath scheduler knobs.
type Config struct {
	MaxChannels       int
	RecalcMinInterval time.Duration
	HysteresisScore   int // min score delta to switch primary
	Cooldown          time.Duration
	BulkWeightCap     int // max weight share per channel (permille)
}

func ConfigForPreset(preset string) Config {
	switch preset {
	case PresetLow:
		return Config{MaxChannels: 3, RecalcMinInterval: 8 * time.Second, HysteresisScore: 120, Cooldown: 45 * time.Second, BulkWeightCap: 450}
	case PresetHigh:
		return Config{MaxChannels: 8, RecalcMinInterval: 3 * time.Second, HysteresisScore: 60, Cooldown: 20 * time.Second, BulkWeightCap: 350}
	default:
		return Config{MaxChannels: 6, RecalcMinInterval: 5 * time.Second, HysteresisScore: 80, Cooldown: 30 * time.Second, BulkWeightCap: 400}
	}
}
