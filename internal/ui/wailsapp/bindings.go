package wailsapp

import (
	"context"
	"fmt"

	"github.com/muhomor/muhomor/internal/ui/presenter"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) requirePres() (*presenter.Presenter, error) {
	if a.pres == nil {
		return nil, fmt.Errorf("GUI не инициализирован")
	}
	return a.pres, nil
}

// GetConnectionSnapshot returns the latest connection DTO.
func (a *App) GetConnectionSnapshot() ConnectionDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pres != nil {
		c, s := a.pres.Snapshot()
		a.lastConn = connectionDTO(c)
		a.lastSet = settingsDTO(s)
	}
	return a.lastConn
}

// GetSettingsSnapshot returns settings for extended tabs.
func (a *App) GetSettingsSnapshot() SettingsDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastSet
}

// Connect starts simple-mode connect.
func (a *App) Connect() string {
	pres, err := a.requirePres()
	if err != nil {
		return err.Error()
	}
	go func() { _ = pres.Connect(a.runCtx) }()
	return ""
}

// Disconnect stops the service.
func (a *App) Disconnect() string {
	pres, err := a.requirePres()
	if err != nil {
		return err.Error()
	}
	go func() { _ = pres.Disconnect(a.runCtx) }()
	return ""
}

// AbortConnect cancels an in-flight connect.
func (a *App) AbortConnect() {
	pres, err := a.requirePres()
	if err != nil {
		return
	}
	go pres.AbortConnect(a.runCtx)
}

// ConnectAction handles connect / disconnect / abort from one button.
func (a *App) ConnectAction() string {
	pres, err := a.requirePres()
	if err != nil {
		return err.Error()
	}
	c, _ := pres.Snapshot()
	switch {
	case c.Busy && !c.Connected:
		a.AbortConnect()
	case c.Connected:
		return a.Disconnect()
	default:
		return a.Connect()
	}
	return ""
}

// ClearPinnedError dismisses a pinned connect/daemon error (see uiKeepErrorsOnScreen).
func (a *App) ClearPinnedError() {
	pres, err := a.requirePres()
	if err != nil {
		return
	}
	pres.ClearPinnedError()
}

// Ping runs latency check (single or bulk pool).
func (a *App) Ping() PingResultDTO {
	pres, err := a.requirePres()
	if err != nil {
		return PingResultDTO{Text: err.Error(), Error: true}
	}
	c, _ := pres.Snapshot()
	if !c.Connected {
		return PingResultDTO{Text: "Нужно подключение", Error: true}
	}
	ctx := a.runCtx
	if ctx == nil {
		ctx = context.Background()
	}
	if len(c.BulkMembers) > 0 {
		resp, e := pres.BulkPingAll(ctx)
		if e != nil {
			return PingResultDTO{Text: e.Error(), Error: true}
		}
		if resp.Error != "" {
			return PingResultDTO{Text: resp.Error, Error: true}
		}
		return PingResultDTO{Text: fmt.Sprintf("%d/%d живых", resp.OK, resp.Total)}
	}
	resp, e := pres.Ping(ctx)
	if e != nil {
		return PingResultDTO{Text: e.Error(), Error: true}
	}
	if resp.Error != "" {
		return PingResultDTO{Text: resp.Error, Error: true}
	}
	if resp.DelayMs > 0 {
		return PingResultDTO{Text: fmt.Sprintf("%d ms", resp.DelayMs)}
	}
	return PingResultDTO{Text: "OK"}
}

// ExportLog exports simple-mode log; returns file path.
func (a *App) ExportLog() (string, error) {
	pres, err := a.requirePres()
	if err != nil {
		return "", err
	}
	ctx := a.runCtx
	if ctx == nil {
		ctx = context.Background()
	}
	return pres.ExportLog(ctx)
}

// ShowWindow restores the main window from tray.
func (a *App) ShowWindow() {
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
		runtime.WindowCenter(a.ctx)
	}
}

// HideWindow hides the main window; daemon and VPN keep running (tray).
func (a *App) HideWindow() {
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}
}

// Quit exits the GUI application (async; see shutdown event for UI).
func (a *App) Quit() {
	a.beginQuit()
}
