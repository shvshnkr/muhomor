package wailsapp

import (
	goruntime "runtime"

	"fyne.io/systray"
)

func setupTray(a *App) func() {
	done := make(chan struct{})
	go func() {
		// Systray HWND + message loop must stay on one OS thread (Wails owns main).
		goruntime.LockOSThread()
		defer goruntime.UnlockOSThread()
		systray.Run(func() { onTrayReady(a) }, func() { close(done) })
	}()
	return func() {
		systray.Quit()
		<-done
	}
}

func onTrayReady(a *App) {
	systray.SetIcon(trayIconBytes())
	systray.SetTooltip("muhomor")
	systray.SetOnTapped(func() { a.ShowWindow() })

	mShow := systray.AddMenuItem("Показать", "Открыть окно")
	mStart := systray.AddMenuItem("Подключить", "Подключить VPN")
	mStop := systray.AddMenuItem("Остановить", "Отключить или отменить")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выход", "Закрыть GUI")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				a.ShowWindow()
			case <-mStart.ClickedCh:
				_ = a.Connect()
			case <-mStop.ClickedCh:
				a.trayStopAction()
			case <-mQuit.ClickedCh:
				a.Quit()
				return
			}
		}
	}()
}

func (a *App) trayStopAction() {
	pres := a.pres
	if pres == nil {
		return
	}
	c, _ := pres.Snapshot()
	if c.Busy && !c.Connected {
		a.AbortConnect()
		return
	}
	_ = a.Disconnect()
}
