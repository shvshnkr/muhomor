package aggregate

import (
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func BenchmarkComputeScore(b *testing.B) {
	in := ChannelInput{ProfileID: 1, RTTMs: 120, GoodputKbps: 8000, LossPermille: 20, JitterMs: 15}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ComputeScore(in)
	}
}

func BenchmarkSchedulerSchedule64(b *testing.B) {
	sch := Scheduler{Config: ConfigForPreset(PresetNormal)}
	profiles := make([]store.Profile, 64)
	inputs := make([]ChannelInput, 64)
	for i := 0; i < 64; i++ {
		id := int64(i + 1)
		profiles[i] = store.Profile{ID: id}
		inputs[i] = ChannelInput{ProfileID: id, RTTMs: 80 + i%40, GoodputKbps: 5000 + i*200}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sch.Schedule(profiles, inputs, FlowBulk, WLPolicy{MaxPct: 25})
	}
}
