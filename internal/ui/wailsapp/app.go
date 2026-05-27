package wailsapp

import (
	"context"
	"strconv"
	"sync"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/paths"
	"github.com/muhomor/muhomor/internal/ui/presenter"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Options mirrors fyneapp launch options.
type Options struct {
	Layout         paths.Layout
	ServiceMode    string
	MixedPort      int
	RouteQuick     int
	DaemonArgs     []string
	StartHidden    bool
	GUILockRelease func() // set by main after AcquireGUILock; nil skips lock in Startup
}

// App is the Wails-bound application struct.
type App struct {
	ctx       context.Context
	runCtx    context.Context
	cancel    context.CancelFunc
	opt       Options
	core      *appcore.App
	pres      *presenter.Presenter
	guiUnlock func()
	trayStop  func()
	mu        sync.Mutex
	quitOnce  sync.Once
	started   bool
	quitReq   bool
	lastConn  ConnectionDTO
	lastSet   SettingsDTO
}

// NewApp builds the Wails app shell (call before wails.Run).
func NewApp(opt Options) *App {
	return &App{opt: opt}
}

// Startup is called by Wails on application start.
func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	if a.started {
		a.ctx = ctx
		a.mu.Unlock()
		return
	}
	a.started = true
	a.mu.Unlock()

	a.ctx = ctx
	if a.cancel != nil {
		a.cancel()
	}
	a.runCtx, a.cancel = context.WithCancel(context.Background())

	if a.opt.GUILockRelease != nil {
		a.guiUnlock = a.opt.GUILockRelease
	}

	api := &apiclient.Client{Dial: apiclient.DialConfig{SocketPath: a.opt.Layout.SocketPath()}}
	core := appcore.NewApp(a.opt.Layout, &appcore.DaemonClient{API: api}, &appcore.RemoteConfig{API: api}, nil)
	core.Groups = &appcore.RemoteGroups{API: api}
	core.Events = &appcore.DaemonEvents{API: api}
	a.core = core

	a.pres = presenter.New(core, a.emitUI)
	a.trayStop = setupTray(a)

	args := a.opt.DaemonArgs
	if len(args) == 0 {
		args = defaultDaemonArgs(a.opt)
	}
	go func() {
		if err := a.pres.Start(a.runCtx, args); err != nil {
			runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
				Title:   "Демон",
				Message: err.Error(),
				Type:    runtime.ErrorDialog,
			})
		}
	}()

	if a.opt.StartHidden {
		runtime.WindowHide(ctx)
	}
}

// Shutdown is called by Wails on application exit.
func (a *App) Shutdown(ctx context.Context) {
	if a.cancel != nil {
		a.cancel()
	}
	if a.trayStop != nil {
		a.trayStop()
	}
	if a.guiUnlock != nil {
		a.guiUnlock()
	}
	_ = ctx
}

// BeforeClose hides to tray instead of quitting (daemon keeps running).
func (a *App) BeforeClose(ctx context.Context) bool {
	a.mu.Lock()
	shouldQuit := a.quitReq
	a.mu.Unlock()
	if shouldQuit {
		return false
	}
	runtime.WindowHide(ctx)
	return true
}

func defaultDaemonArgs(opt Options) []string {
	var args []string
	if opt.ServiceMode != "" {
		args = append(args, "--service-mode", opt.ServiceMode)
	}
	if opt.MixedPort > 0 {
		args = append(args, "--mixed-port", strconv.Itoa(opt.MixedPort))
	}
	if opt.RouteQuick >= 0 {
		args = append(args, "--route-quick-profile", strconv.Itoa(opt.RouteQuick))
	}
	return args
}
