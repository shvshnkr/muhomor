package wailsapp

import (
	"github.com/muhomor/muhomor/internal/ui/model"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const eventConnection = "connection"
const eventSettings = "settings"

func (a *App) emitUI(c model.ConnectionUI, s model.SettingsUI) {
	conn := connectionDTO(c)
	set := settingsDTO(s)
	a.mu.Lock()
	if a.quitReq {
		conn = sanitizeConnectionForQuit(conn)
	}
	a.lastConn = conn
	a.lastSet = set
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, eventConnection, conn)
		runtime.EventsEmit(a.ctx, eventSettings, set)
	}
}
