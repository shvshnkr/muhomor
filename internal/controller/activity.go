package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/muhomor/muhomor/internal/mihomo"
	"github.com/muhomor/muhomor/internal/store"
)

func (r *Runtime) startMihomoClient(ctx context.Context, cfgPath string, reloadIfRunning bool) (*mihomo.Client, error) {
	if r.lifecycle == nil {
		r.lifecycle = NewLifecycleSupervisor()
	}
	var out *mihomo.Client
	err := r.lifecycle.Do(LifecycleStarting, func() error {
		r.mu.Lock()
		existing := r.mihomo
		r.mu.Unlock()

		if reloadIfRunning && existing != nil {
			if err := existing.Reload(ctx); err == nil {
				r.markMihomoReload()
				out = existing
				return nil
			}
		}

		if existing != nil {
			existing.Stop()
		}

		secret := r.build.Secret
		if s := mihomo.SecretFromConfig(cfgPath); s != "" {
			secret = s
		}
		client := mihomo.NewClient(mihomo.ClientOptions{
			ConfigPath: cfgPath,
			ConfigDir:  r.Paths.MihomoDir(),
			Controller: r.build.ExternalController,
			Secret:     secret,
		})
		if err := client.Start(ctx); err != nil {
			return fmt.Errorf("mihomo start: %w", err)
		}
		r.mu.Lock()
		r.mihomo = client
		r.mu.Unlock()
		r.markMihomoReload()
		out = client
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Runtime) markMihomoReload() {
	r.mu.Lock()
	r.mihomoReloadAt = time.Now()
	r.mu.Unlock()
}

func (r *Runtime) setActivity(ctx context.Context, text string) {
	_ = r.Store.SetKV(ctx, store.KeySimpleModeActivity, text)
	if text != "" {
		_ = os.MkdirAll(r.Paths.CacheDir, 0o700)
		path := filepath.Join(r.Paths.CacheDir, "activity.log")
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d %s\n", time.Now().UnixMilli(), text)
			_ = f.Close()
		}
	}
	r.mu.Lock()
	if text != "" && r.status.State != StateConnected {
		r.status.State = StateConnecting
	}
	r.mu.Unlock()
	r.publishStatusEvent("activity")
}

func (r *Runtime) clearActivity(ctx context.Context) {
	r.setActivity(ctx, "")
}
