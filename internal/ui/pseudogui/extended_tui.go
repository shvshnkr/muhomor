package pseudogui

import (
	"context"
	"fmt"
	"strings"

	"github.com/muhomor/muhomor/internal/ui/console"
)

type extendedTab int

const (
	tabSimple extendedTab = iota
	tabConfig
	tabRoute
	tabSettings
)

var extendedTabLabels = []string{
	"Простой",
	"Конфигурация",
	"Маршрут",
	"Настройки",
}

// RunExtendedTUI shell (sidebar like Wails Extended.tsx).
func RunExtendedTUI(ctx context.Context, cfg Config) int {
	io := cfg.IO
	if io == nil {
		io = console.StdIO()
	}
	term := console.NewTerminal(io)
	if err := term.EnterRaw(); err != nil {
		return 0
	}
	defer term.LeaveRaw()

	tab := tabConfig
	if cfg.Pres != nil {
		tab = tabSimple
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

	var toast string
	lastFrame := ""
	paint := func() {
		frame := renderExtendedScreen(tab, toast)
		if frame == lastFrame {
			return
		}
		if lastFrame == "" {
			term.ClearScreen()
		} else {
			term.DrawHome(frame)
		}
		fmt.Fprint(term.Out, frame)
		lastFrame = frame
	}
	paint()

	for {
		select {
		case <-ctx.Done():
			return 0
		case <-errc:
			return 0
		case k := <-keys:
			switch k {
			case console.KeyQuit:
				return 0
			case console.KeyEsc:
				return 0
			case console.KeyUp:
				if tab == 0 {
					tab = extendedTab(len(extendedTabLabels) - 1)
				} else {
					tab--
				}
				paint()
			case console.KeyDown:
				if tab >= extendedTab(len(extendedTabLabels)-1) {
					tab = 0
				} else {
					tab++
				}
				paint()
			case console.KeyEnter:
				switch tab {
				case tabSimple:
					return 0
				case tabConfig, tabRoute, tabSettings:
					term.LeaveRaw()
					code := openExtendedPanel(ctx, cfg, tab)
					_ = term.EnterRaw()
					if code == 0 {
						toast = "Готово"
					}
					lastFrame = ""
					paint()
				}
			}
		}
	}
}

func renderExtendedScreen(tab extendedTab, toast string) string {
	var b strings.Builder
	b.WriteString(ansiBold + "muhomor — расширенный режим" + ansiReset + "\n\n")
	for i, label := range extendedTabLabels {
		selected := extendedTab(i) == tab
		line := menuRow(label, selected, false, 36)
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
	switch tab {
	case tabSimple:
		b.WriteString(stubPanel("Простой", "Esc — главный экран с кнопкой подключения.\nТот же simple mode, что в muhomor-gui."))
	case tabConfig:
		b.WriteString(stubPanel("Конфигурация", "Enter — меню групп и подписок [g].\nПолная вкладка — в muhomor-gui → Конфигурация."))
	case tabRoute:
		b.WriteString(stubPanel("Маршрут", "Enter — выбор пресета (↑↓ или 0–3).\nПолная вкладка — в muhomor-gui → Маршрут."))
	case tabSettings:
		b.WriteString(stubPanel("Настройки", "Enter — расширенные KV [s].\nПолная вкладка — в muhomor-gui → Настройки."))
	}
	if toast != "" {
		b.WriteString("\n" + ansiTeal + toast + ansiReset + "\n")
	}
	b.WriteString(ansiDim + "\n↑↓ — вкладка   Enter — открыть   Esc — назад   q — выход" + ansiReset + "\n")
	return padFrameLines(b.String(), 20)
}

func stubPanel(title, body string) string {
	return ansiBold + title + ansiReset + "\n\n" + body + "\n"
}

func openExtendedPanel(ctx context.Context, cfg Config, tab extendedTab) int {
	switch tab {
	case tabConfig:
		return RunGroupsMenu(ctx, cfg.App, cfg.IO)
	case tabRoute:
		return RunRouteMenu(ctx, cfg.App, cfg.IO)
	case tabSettings:
		return RunSettingsMenu(ctx, cfg.App, cfg.IO)
	default:
		return 0
	}
}
