package pseudogui

import (
	"context"
	"fmt"
	"time"

	"github.com/muhomor/muhomor/internal/ui/console"
	"github.com/muhomor/muhomor/internal/ui/model"
	"github.com/muhomor/muhomor/internal/ui/presenter"
)

type sessionTimer struct {
	start  time.Time
	active bool
}

func (t *sessionTimer) tick(connected bool) int {
	if connected {
		if !t.active {
			t.start = time.Now()
			t.active = true
		}
		return int(time.Since(t.start).Seconds())
	}
	t.active = false
	return 0
}

// RunSimpleTUI is the main pseudo-GUI screen (parity with Wails Simple.tsx).
func RunSimpleTUI(ctx context.Context, cfg Config) int {
	io := cfg.IO
	if io == nil {
		io = console.StdIO()
	}
	term := console.NewTerminal(io)
	if err := term.EnterRaw(); err != nil {
		io.Line("TUI: " + err.Error() + " — нужен интерактивный терминал")
		return RunLegacyMenu(ctx, cfg)
	}
	defer term.LeaveRaw()

	term.UseAltScreen()
	defer term.RestoreAltScreen()

	pres := cfg.Pres
	var clock sessionTimer
	v := simpleView{focus: focusConnect}

	uiNotify := make(chan struct{}, 4)
	if pres != nil {
		pres.OnUI = func(c model.ConnectionUI, s model.SettingsUI) {
			v.conn = c
			v.set = s
			select {
			case uiNotify <- struct{}{}:
			default:
			}
		}
	}

	keys := make(chan console.Key, 8)
	errc := make(chan error, 1)
	go func() {
		for {
			k, err := term.ReadKey()
			if err != nil {
				errc <- err
				return
			}
			keys <- k
		}
	}()

	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	poll := time.NewTicker(2 * time.Second)
	defer poll.Stop()

	refresh := func() {
		if pres != nil {
			_ = pres.Refresh(ctx)
			c, s := pres.Snapshot()
			v.conn = c
			v.set = s
			v.sessionSec = clock.tick(c.Connected)
			syncSimpleToast(&v)
		}
	}
	refresh()

	lastFrame := ""
	paint := func(force bool) {
		frame := renderSimpleScreen(v, false)
		if !force && frame == lastFrame {
			return
		}
		if lastFrame == "" {
			term.ClearScreen()
			fmt.Fprint(term.Out, frame)
		} else {
			term.DrawHome(frame)
		}
		lastFrame = frame
	}
	paint(true)

	for {
		select {
		case <-ctx.Done():
			return 0
		case err := <-errc:
			if err != nil {
				term.LeaveRaw()
				io.Line("")
				return 0
			}
		case <-uiNotify:
			if pres != nil {
				c, s := pres.Snapshot()
				v.conn = c
				v.set = s
				syncSimpleToast(&v)
			}
			paint(false)
		case <-tick.C:
			if v.conn.Connected {
				v.sessionSec = clock.tick(true)
				paint(false)
			}
		case <-poll.C:
			if v.conn.Connected || uiConnecting(v.conn) {
				refresh()
				paint(false)
			}
		case k := <-keys:
			if uiConnecting(v.conn) {
				switch k {
				case console.KeyQuit, console.KeyEsc, console.KeyEnter:
					if pres != nil {
						pres.AbortConnect(ctx)
					}
					refresh()
					paint(true)
				}
				continue
			}
			switch k {
			case console.KeyQuit, console.KeyEsc:
				return 0
			case console.KeyUp:
				v.focus = prevFocus(v.focus, v.conn)
				paint(true)
			case console.KeyDown:
				v.focus = nextFocus(v.focus, v.conn)
				paint(true)
			case console.KeyEnter:
				mode := activateFocus(ctx, cfg, &v, keys, paint, refresh)
				if mode == uiModeExtended {
					term.LeaveRaw()
					_ = RunExtendedTUI(ctx, cfg)
					_ = term.EnterRaw()
					term.UseAltScreen()
					refresh()
					lastFrame = ""
					paint(true)
				} else {
					paint(true)
				}
			}
		}
	}
}

func prevFocus(f simpleFocus, c model.ConnectionUI) simpleFocus {
	for {
		if f == 0 {
			return focusExtended
		}
		f--
		if f == focusPing && !c.Connected && !uiConnecting(c) {
			continue
		}
		if f == focusDiagnostics && probeBlock(c) == "" {
			continue
		}
		return f
	}
}

func nextFocus(f simpleFocus, c model.ConnectionUI) simpleFocus {
	for {
		if f >= focusExtended {
			return focusConnect
		}
		f++
		if f == focusPing && !c.Connected && !uiConnecting(c) {
			continue
		}
		if f == focusDiagnostics && probeBlock(c) == "" {
			continue
		}
		return f
	}
}

type uiMode int

const (
	uiModeSimple uiMode = iota
	uiModeExtended
)

func activateFocus(
	ctx context.Context,
	cfg Config,
	v *simpleView,
	keys <-chan console.Key,
	paint func(bool),
	refresh func(),
) uiMode {
	switch v.focus {
	case focusConnect:
		runConnectWithUI(ctx, cfg, v, keys, paint, refresh)
		return uiModeSimple
	case focusPing:
		if !v.conn.Connected || cfg.Pres == nil {
			v.toast, v.toastErr = "Сначала подключитесь", false
			return uiModeSimple
		}
		v.pingBusy = true
		paint(true)
		_ = doPing(ctx, cfg.Pres, v)
		v.pingBusy = false
		return uiModeSimple
	case focusDiagnostics:
		if probeBlock(v.conn) == "" {
			return uiModeSimple
		}
		v.diagOpen = !v.diagOpen
		return uiModeSimple
	case focusExport:
		_ = doExport(ctx, cfg, v)
		return uiModeSimple
	case focusExtended:
		return uiModeExtended
	default:
		return uiModeSimple
	}
}

func runConnectWithUI(
	ctx context.Context,
	cfg Config,
	v *simpleView,
	keys <-chan console.Key,
	paint func(bool),
	refresh func(),
) {
	if cfg.Pres == nil {
		_ = runAction(ctx, cfg.App, cfg.Pres, cfg.IO, ActionStart, cfg.DaemonArgs)
		refresh()
		return
	}
	c, _ := cfg.Pres.Snapshot()
	if c.Connected {
		_ = cfg.Pres.Disconnect(ctx)
		syncSimpleToast(v)
		refresh()
		return
	}

	done := make(chan error, 1)
	go func() { done <- cfg.Pres.Connect(ctx) }()

	uiTick := time.NewTicker(120 * time.Millisecond)
	defer uiTick.Stop()

	for {
		c, s := cfg.Pres.Snapshot()
		v.conn = c
		v.set = s
		paint(true)

		select {
		case err := <-done:
			_ = err
			syncSimpleToast(v)
			refresh()
			return
		case <-uiTick.C:
		case k := <-keys:
			switch k {
			case console.KeyEnter, console.KeyEsc:
				cfg.Pres.AbortConnect(ctx)
			case console.KeyQuit:
				cfg.Pres.AbortConnect(ctx)
				return
			}
		}
	}
}

func doExport(ctx context.Context, cfg Config, v *simpleView) int {
	if cfg.Pres == nil {
		err := cfg.App.RunCtlCommand(ctx, "export-log")
		if err != nil {
			v.toast, v.toastErr = err.Error(), true
			return 1
		}
		v.toast = "Лог экспортирован"
		return 0
	}
	path, err := cfg.Pres.ExportLog(ctx)
	if err != nil {
		v.toast, v.toastErr = err.Error(), true
		return 1
	}
	v.toast = "Лог: " + path
	return 0
}

func doPing(ctx context.Context, pres *presenter.Presenter, v *simpleView) int {
	c, _ := pres.Snapshot()
	if len(c.BulkMembers) > 0 && c.Connected {
		resp, err := pres.BulkPingAll(ctx)
		if err != nil {
			v.toast, v.toastErr = err.Error(), true
			return 1
		}
		if resp.Error != "" {
			v.toast, v.toastErr = resp.Error, true
		} else {
			v.toast = fmt.Sprintf("Пинг: %d/%d живых", resp.OK, resp.Total)
		}
	} else {
		resp, err := pres.Ping(ctx)
		if err != nil {
			v.toast, v.toastErr = err.Error(), true
			return 1
		}
		if resp.Error != "" {
			v.toast, v.toastErr = resp.Error, true
		} else if resp.DelayMs > 0 {
			v.toast = fmt.Sprintf("%d ms", resp.DelayMs)
		}
	}
	nc, ns := pres.Snapshot()
	v.conn, v.set = nc, ns
	return 0
}
