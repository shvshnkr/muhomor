package wailsapp

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

const windowTitle = "muhomor"

// DesktopRunOptions is passed to RunDesktop after CLI parsing and GUI lock.
type DesktopRunOptions struct {
	App         *App
	Assets      embed.FS
	StartHidden bool // hide main window on startup (tray-only smoke)
}

// RunDesktop starts the Wails desktop shell (frameless + custom WindowChrome in frontend).
func RunDesktop(opt DesktopRunOptions) error {
	return wails.Run(&options.App{
		Title:             windowTitle,
		Width:             420,
		Height:            552,
		MinWidth:          380,
		MinHeight:         440,
		DisableResize:     false,
		Frameless:         true,
		StartHidden:       opt.StartHidden,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 9, G: 11, B: 15, A: 255},
		OnStartup:         opt.App.Startup,
		OnShutdown:        opt.App.Shutdown,
		OnBeforeClose:     opt.App.BeforeClose,
		Bind: []interface{}{
			opt.App,
		},
		AssetServer: &assetserver.Options{
			Assets: opt.Assets,
		},
		Windows: &windows.Options{
			Theme:                            windows.Dark,
			WebviewIsTransparent:             false,
			WindowIsTranslucent:              false,
			DisableFramelessWindowDecorations: true,
			CustomTheme: &windows.ThemeSettings{
				DarkModeBorder:         0x000F0B09,
				DarkModeBorderInactive: 0x000F0B09,
			},
		},
	})
}

// WindowTitle is used for single-instance activation (platform.ActivateGUIWindow).
func WindowTitle() string { return windowTitle }
