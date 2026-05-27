package wailsapp

import (
	"context"
	"time"

	"github.com/muhomor/muhomor/internal/platform"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ShutdownDTO is pushed to the frontend during application exit.
type ShutdownDTO struct {
	Active  bool   `json:"active"`
	Phase   string `json:"phase"`
	Message string `json:"message"`
}

const eventShutdown = "shutdown"

func (a *App) emitShutdown(d ShutdownDTO) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, eventShutdown, d)
	}
}

// Quit begins async shutdown (daemon stop if configured) and exits when done.
// Safe to call multiple times; only the first call runs the shutdown sequence.
func (a *App) beginQuit() {
	a.quitOnce.Do(func() {
		a.mu.Lock()
		a.quitReq = true
		a.mu.Unlock()

		// Overlay before cancel/SSE teardown so the UI never flashes daemon errors.
		a.emitShutdown(ShutdownDTO{
			Active:  true,
			Phase:   "start",
			Message: "Завершение…",
		})
		if a.ctx != nil {
			runtime.WindowShow(a.ctx)
		}
		if a.pres != nil {
			a.pres.SetShuttingDown(true)
		}
		go a.runQuitSequence()
	})
}

func (a *App) runQuitSequence() {
	// Let the webview paint the shutdown overlay before tearing down SSE/API.
	time.Sleep(120 * time.Millisecond)

	if a.cancel != nil {
		a.cancel()
	}

	stopDaemon := false
	if a.core != nil && a.core.Config != nil {
		loadCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		set, err := a.core.Config.LoadSettings(loadCtx)
		cancel()
		if err == nil && set.StopDaemonOnExit {
			stopDaemon = true
		}
	}

	if stopDaemon {
		a.emitShutdown(ShutdownDTO{
			Active:  true,
			Phase:   "daemon",
			Message: "Остановка демона…",
		})
		stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		_ = platform.StopDaemon(stopCtx, a.opt.Layout)
		cancel()
	}

	a.emitShutdown(ShutdownDTO{
		Active:  true,
		Phase:   "exit",
		Message: "Закрытие…",
	})

	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}
