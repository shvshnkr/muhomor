package controller

import (
	"context"
	"fmt"

	"github.com/muhomor/muhomor/internal/mihomo"
	"github.com/muhomor/muhomor/internal/store"
)

func (r *Runtime) startMihomoClient(ctx context.Context, cfgPath string) (*mihomo.Client, error) {
	r.mu.Lock()
	if r.mihomo != nil {
		r.mihomo.Stop()
		r.mihomo = nil
	}
	r.mu.Unlock()
	mihomo.KillAll()

	client := mihomo.NewClient(mihomo.ClientOptions{
		ConfigPath: cfgPath,
		ConfigDir:  r.Paths.MihomoDir(),
		Controller: r.build.ExternalController,
		Secret:     r.build.Secret,
	})
	if err := client.Start(ctx); err != nil {
		return nil, fmt.Errorf("mihomo start: %w", err)
	}
	r.mu.Lock()
	r.mihomo = client
	r.mu.Unlock()
	return client, nil
}

func (r *Runtime) setActivity(ctx context.Context, text string) {
	_ = r.Store.SetKV(ctx, store.KeySimpleModeActivity, text)
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
