package controller

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/muhomor/muhomor/internal/mihomo"
)

const trafficSampleTTL = time.Second
const (
	trafficMaxBytesPerSec     int64         = 625_000_000 // 5 Gbps.
	trafficSpikeRatioLimit    int64         = 20
	trafficSpikeFloorBytesSec int64         = 50_000_000
	trafficErrorStaleAfter    int           = 3
	trafficErrorLogInterval   time.Duration = 10 * time.Second
	trafficReloadGrace                    = 2 * time.Second
)

func shouldLogTrafficUnitFallback(rawUp, rawDown int64, upMode, downMode string) bool {
	if rawUp == 0 && rawDown == 0 {
		return false
	}
	maxRaw := rawUp
	if rawDown > maxRaw {
		maxRaw = rawDown
	}
	if maxRaw < 1024 {
		return false
	}
	if upMode == "zero" && downMode == "zero" {
		return false
	}
	return upMode != "kbps" || downMode != "kbps"
}

func isTrafficReloadNoise(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "forcibly closed") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection was aborted") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "eof")
}

func (r *Runtime) skipTrafficSample() bool {
	if r.connectInFlight() {
		return true
	}
	if r.Status().State == StateConnecting {
		return true
	}
	r.mu.Lock()
	at := r.mihomoReloadAt
	r.mu.Unlock()
	return !at.IsZero() && time.Since(at) < trafficReloadGrace
}

// mihomo GET /traffic reports rates in kbps (kilobits per second).
func kbpsToBytesPerSec(kbps int64) int64 {
	if kbps <= 0 {
		return 0
	}
	return kbps * 1000 / 8
}

func (r *Runtime) cachedTraffic() (up, down int64) {
	r.trafficMu.Lock()
	defer r.trafficMu.Unlock()
	return r.trafficUp, r.trafficDown
}

func normalizeTrafficRate(raw int64) (value int64, mode string, err error) {
	if raw < 0 {
		return 0, "", fmt.Errorf("negative rate: %d", raw)
	}
	if raw == 0 {
		return 0, "kbps", nil
	}
	asKbps := kbpsToBytesPerSec(raw)
	if asKbps <= trafficMaxBytesPerSec {
		return asKbps, "kbps", nil
	}
	if raw <= trafficMaxBytesPerSec {
		return raw, "bytes_per_sec", nil
	}
	return 0, "", fmt.Errorf("rate too high: raw=%d", raw)
}

func ratioClose(observed, expected int64) bool {
	if observed <= 0 || expected <= 0 {
		return false
	}
	lo := float64(expected) * 0.5
	hi := float64(expected) * 1.5
	v := float64(observed)
	return v >= lo && v <= hi
}

func normalizeTrafficRateWithTotals(raw, deltaTotal int64, dt time.Duration) (value int64, mode string, err error) {
	if raw < 0 {
		return 0, "", fmt.Errorf("negative rate: %d", raw)
	}
	if raw == 0 {
		return 0, "zero", nil
	}
	if dt <= 0 {
		return normalizeTrafficRate(raw)
	}
	sec := dt.Seconds()
	if sec <= 0 {
		return normalizeTrafficRate(raw)
	}
	estimatedBps := int64(math.Round(float64(deltaTotal) / sec))
	asKbps := kbpsToBytesPerSec(raw)
	switch {
	case ratioClose(estimatedBps, raw):
		return raw, "bytes_per_sec", nil
	case ratioClose(estimatedBps, asKbps):
		return asKbps, "kbps", nil
	default:
		return normalizeTrafficRate(raw)
	}
}

func sanitizeTrafficSample(prevUp, prevDown, up, down int64) (int64, int64) {
	if prevUp > 0 && up > prevUp*trafficSpikeRatioLimit && up >= trafficSpikeFloorBytesSec {
		up = prevUp
	}
	if prevDown > 0 && down > prevDown*trafficSpikeRatioLimit && down >= trafficSpikeFloorBytesSec {
		down = prevDown
	}
	return up, down
}

func (r *Runtime) trafficSampleError(err error) {
	if r.skipTrafficSample() && isTrafficReloadNoise(err) {
		return
	}
	now := time.Now()
	r.trafficMu.Lock()
	r.trafficErrs++
	count := r.trafficErrs
	lastLog := r.trafficErrAt
	shouldStale := count >= trafficErrorStaleAfter
	if shouldStale {
		r.trafficUp, r.trafficDown = 0, 0
		r.trafficAt = time.Time{}
	}
	logNow := lastLog.IsZero() || now.Sub(lastLog) >= trafficErrorLogInterval
	if logNow {
		r.trafficErrAt = now
	}
	r.trafficMu.Unlock()
	if logNow && r.Log != nil {
		r.Log.Warn("traffic sample failed", "err", err, "errors", count, "stale", shouldStale)
	}
}

func (r *Runtime) refreshTraffic(ctx context.Context, client *mihomo.Client) {
	if client == nil || r.skipTrafficSample() {
		return
	}
	tctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	snap, err := client.TrafficSnapshotNow(tctx)
	if err != nil {
		r.trafficSampleError(err)
		return
	}
	rawUp, rawDown := snap.Up, snap.Down
	r.trafficMu.Lock()
	prevAt := r.trafficAt
	prevUpTotal := r.trafficUpTotal
	prevDownTotal := r.trafficDownTotal
	r.trafficMu.Unlock()
	dt := time.Since(prevAt)
	deltaUpTotal := snap.UpTotal - prevUpTotal
	deltaDownTotal := snap.DownTotal - prevDownTotal
	if deltaUpTotal < 0 {
		deltaUpTotal = 0
	}
	if deltaDownTotal < 0 {
		deltaDownTotal = 0
	}
	up, upMode, err := normalizeTrafficRateWithTotals(rawUp, deltaUpTotal, dt)
	if err != nil {
		r.trafficSampleError(fmt.Errorf("up: %w", err))
		return
	}
	down, downMode, err := normalizeTrafficRateWithTotals(rawDown, deltaDownTotal, dt)
	if err != nil {
		r.trafficSampleError(fmt.Errorf("down: %w", err))
		return
	}
	r.trafficMu.Lock()
	if r.trafficErrs > 0 {
		r.trafficErrs = 0
	}
	prevUp, prevDown := r.trafficUp, r.trafficDown
	up, down = sanitizeTrafficSample(prevUp, prevDown, up, down)
	r.trafficUp, r.trafficDown = up, down
	r.trafficUpTotal, r.trafficDownTotal = snap.UpTotal, snap.DownTotal
	r.trafficAt = time.Now()
	r.trafficMu.Unlock()
	if shouldLogTrafficUnitFallback(rawUp, rawDown, upMode, downMode) && r.Log != nil {
		r.Log.Warn("traffic unit fallback", "up_mode", upMode, "down_mode", downMode, "raw_up", rawUp, "raw_down", rawDown)
	}
}

func (r *Runtime) clearTrafficCache() {
	r.trafficMu.Lock()
	r.trafficUp, r.trafficDown = 0, 0
	r.trafficUpTotal, r.trafficDownTotal = 0, 0
	r.trafficAt = time.Time{}
	r.trafficErrs = 0
	r.trafficErrAt = time.Time{}
	r.trafficMu.Unlock()
}

func (r *Runtime) startTrafficSampler(ctx context.Context) {
	r.trafficMu.Lock()
	if r.trafficStop != nil {
		r.trafficMu.Unlock()
		return
	}
	stop := make(chan struct{})
	r.trafficStop = stop
	r.trafficMu.Unlock()

	go func() {
		tick := time.NewTicker(trafficSampleTTL)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-tick.C:
				r.mu.Lock()
				client := r.mihomo
				proxy := r.proxy
				r.mu.Unlock()
				if client == nil || proxy == "" {
					continue
				}
				r.refreshTraffic(ctx, client)
			}
		}
	}()
}

func (r *Runtime) stopTrafficSampler() {
	r.trafficMu.Lock()
	stop := r.trafficStop
	r.trafficStop = nil
	r.trafficMu.Unlock()
	if stop != nil {
		close(stop)
	}
}
