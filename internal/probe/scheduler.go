package probe

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/store"
)

// Scheduler maintains background TCP probe state in DB (feature-flagged).
type Scheduler struct {
	Store  *store.Store
	Log    *slog.Logger
	Config Config
}

func (sc *Scheduler) Run(ctx context.Context) {
	if sc.Store == nil || !sc.Store.ProbeSchedulerEnabled(ctx) {
		return
	}
	cfg := sc.Config
	if cfg.TickInterval == 0 {
		cfg = DefaultConfig()
	}
	tick := time.NewTicker(cfg.TickInterval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			sc.runTick(ctx, cfg)
		}
	}
}

func (sc *Scheduler) runTick(ctx context.Context, cfg Config) {
	now := time.Now()
	profiles, err := sc.Store.ListProfilesDueProbe(ctx, cfg.TCPBatchPerTick, now)
	if err != nil || len(profiles) == 0 {
		return
	}
	var checked, ok, fail int
	sem := make(chan struct{}, cfg.TCPWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, p := range profiles {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			meta, err := sc.Store.ProbeMetaByID(ctx, p.ID)
			if err != nil {
				meta = store.ProbeMeta{State: store.ProbeUnknown, SourcePriority: store.ProbeSourceSubscription}
			}
			if p.WLBuiltinPool {
				meta.SourcePriority = store.ProbeSourceBuiltin
			}
			delay, errClass, tcpMs := sc.tcpProbe(ctx, p, cfg.TCPTimeout)
			meta.LastCheckedAt = now
			if tcpMs > 0 {
				ok++
				meta.FailStreak = 0
				meta.SuccessWindow++
				meta.LastOKAt = now
				meta.EWMADelayMs = ewma(meta.EWMADelayMs, tcpMs)
				meta.State = store.ProbeCandidate
				if delay > 0 {
					meta.State = store.ProbeAlive
				}
				meta.LastErrorClass = ""
				meta.NextProbeAt = now.Add(cfg.SuspectRetry * 4)
			} else {
				fail++
				meta.FailStreak++
				meta.LastFailAt = now
				meta.LastErrorClass = errClass
				if meta.FailStreak >= 6 {
					meta.State = store.ProbeCemetery
				} else if meta.FailStreak >= 2 {
					meta.State = store.ProbeDead
				} else {
					meta.State = store.ProbeSuspect
				}
				meta.NextProbeAt = NextProbeAfter(meta.State, meta.FailStreak, now)
			}
			mu.Lock()
			checked++
			mu.Unlock()
			_ = sc.Store.UpdateProfileProbeMeta(ctx, p.ID, meta)
		}()
	}
	wg.Wait()
	counts, _ := sc.Store.CountProbeStates(ctx)
	total, _ := sc.Store.CountEnabledProfiles(ctx)
	preset := sc.Store.ProbePreset(ctx)
	reason, _ := sc.Store.GetKV(ctx, store.KeyProbeLastSelectReason)
	_ = sc.Store.SetProbeStats(ctx, store.ProbeStats{
		TotalEnabled:      total,
		Unknown:           counts[store.ProbeUnknown],
		Candidate:         counts[store.ProbeCandidate],
		Alive:             counts[store.ProbeAlive],
		Suspect:           counts[store.ProbeSuspect],
		Dead:              counts[store.ProbeDead],
		Cemetery:          counts[store.ProbeCemetery],
		LastTickChecked:   checked,
		LastTickOK:        ok,
		LastTickFail:      fail,
		SchedulerEnabled:  true,
		WarmSelectEnabled: sc.Store.ProbeWarmSelectEnabled(ctx),
		Preset:            preset,
		LastSelectReason:  reason,
	})
	if sc.Log != nil && checked > 0 {
		sc.Log.Info("probe scheduler tick",
			"checked", checked, "ok", ok, "fail", fail,
			"alive", counts[store.ProbeAlive], "event", "2K-probe")
	}
}

func (sc *Scheduler) tcpProbe(ctx context.Context, p store.Profile, timeout time.Duration) (delay int, errClass string, tcpMs int) {
	host, port, err := profileHostPort(p)
	if err != nil || host == "" {
		return 0, "parse", 0
	}
	start := time.Now()
	d := net.Dialer{Timeout: timeout}
	c, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
	if err != nil {
		return 0, "tcp_dial", 0
	}
	_ = c.Close()
	tcpMs = int(time.Since(start).Milliseconds())
	return tcpMs, "", tcpMs
}

func ewma(prev, sample int) int {
	if prev <= 0 {
		return sample
	}
	return (prev*7 + sample*3) / 10
}

func profileHostPort(p store.Profile) (string, int, error) {
	switch p.Type {
	case "vless", "":
		v, e := configgen.ParseVLESSURI(p.URI)
		return v.Server, v.Port, e
	case "trojan":
		t, e := configgen.ParseTrojanURI(p.URI)
		return t.Server, t.Port, e
	case "hysteria", "hysteria2":
		h, e := configgen.ParseHysteriaURI(p.URI)
		return h.Server, h.Port, e
	default:
		return "", 0, fmt.Errorf("unsupported")
	}
}
