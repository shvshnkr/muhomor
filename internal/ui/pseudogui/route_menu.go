package pseudogui

import (
	"context"

	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/ui/console"
)

var routeQuickOptions = []string{
	"Ручной",
	"RU напрямую",
	"RU/заблокированное и AI через прокси",
	"WG over WL tunnel (private через прокси)",
}

// RunRouteMenu DOS-style route preset picker (0..3).
func RunRouteMenu(ctx context.Context, app *appcore.App, io *console.IO) int {
	current := 0
	if v, err := app.Config.RouteQuickProfile(ctx); err == nil && v >= 0 && v < len(routeQuickOptions) {
		current = v
	}
	idx, ok := RunListPicker(ctx, io, "Быстрый маршрут", routeQuickOptions, current)
	if !ok {
		return 0
	}
	if err := app.SetRouteQuick(ctx, idx); err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	io.Line("Маршрут: " + routeQuickOptions[idx] + " — применится при connect/reload")
	return 0
}
