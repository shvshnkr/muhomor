package pseudogui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/ui/console"
	"github.com/muhomor/muhomor/internal/ui/presenter"
)

// Config for pseudo-GUI session.
type Config struct {
	App        *appcore.App
	IO         *console.IO
	Pres       *presenter.Presenter
	DaemonArgs []string
}

// Run starts the arrow-key TUI (Simple screen); falls back to legacy menu if not a TTY.
func Run(ctx context.Context, cfg Config) int {
	return RunSimpleTUI(ctx, cfg)
}

// RunLegacyMenu is the numbered Dahusim-style menu (fallback).
func RunLegacyMenu(ctx context.Context, cfg Config) int {
	app := cfg.App
	io := cfg.IO
	pres := cfg.Pres
	if io == nil {
		io = console.StdIO()
	}

	io.Line("")
	io.Line("muhomor — простой режим (терминал)")
	io.Line("Прокси: curl -x http://127.0.0.1:<mixed-port> …")

	for {
		if pres != nil {
			_ = pres.Refresh(ctx)
			c, s := pres.Snapshot()
			printStatusBanner(io, c, s)
		}
		printMenu(io)
		choice, err := io.ReadLine("Команда: ")
		if err != nil {
			return 0
		}
		action, ok := ParseChoice(choice)
		if !ok {
			io.Line("Неизвестная команда — см. меню")
			continue
		}
		if action == ActionQuit {
			return 0
		}
		if code := runAction(ctx, app, pres, io, action, cfg.DaemonArgs); code != 0 {
			io.Line(fmt.Sprintf("Ошибка (код %d)", code))
		}
		io.Line("")
	}
}

func printMenu(io *console.IO) {
	io.Line("")
	io.Line("── Подключение ──")
	io.Line("  [3] Подключить / отключить")
	io.Line("  [2] Ping (весь пул, если load-balance)")
	io.Line("  [1] Подробный статус")
	io.Line("  [b] Пул load-balance: ноги и задержки")
	io.Line("── Сервис ──")
	io.Line("  [4] Остановить   [5] Перезагрузить   [6] Экспорт лога")
	io.Line("  [7] Проверить обновление   [8] Установить")
	io.Line("── Данные ──")
	io.Line("  [p] Профили   [i] Импорт URI   [h] Цепочка")
	io.Line("  [g] Группы (подписки)")
	io.Line("── Настройки ──")
	io.Line("  [m] Proxy/VPN   [r] Маршрут 0–3   [s] Настройки")
	io.Line("  [a] Адаптация сети   [d] Демон")
	io.Line("  [q] Выход")
}

func runAction(ctx context.Context, app *appcore.App, pres *presenter.Presenter, io *console.IO, action MenuAction, daemonArgs []string) int {
	var err error
	switch action {
	case ActionStatus:
		if pres != nil {
			err = pres.Refresh(ctx)
			if err == nil {
				c, s := pres.Snapshot()
				printStatus(io, c, s)
			}
		} else {
			err = app.RunCtlCommand(ctx, "status")
		}
	case ActionPing:
		err = runPing(ctx, app, pres, io)
	case ActionBulkPool:
		if pres != nil {
			err = pres.Refresh(ctx)
			if err == nil {
				c, _ := pres.Snapshot()
				printBulkMembers(io, c)
			}
		} else {
			io.Line("нужен демон с presenter")
			return 1
		}
	case ActionStart:
		if pres == nil {
			err = app.RunCtlCommand(ctx, "start")
			break
		}
		c, _ := pres.Snapshot()
		if c.Connected {
			io.Line("Отключение…")
			err = pres.Disconnect(ctx)
		} else {
			io.Line("Подключение…")
			err = pres.Connect(ctx)
		}
		if err == nil {
			c, s := pres.Snapshot()
			printStatusBanner(io, c, s)
		}
	case ActionStop:
		if pres != nil {
			err = pres.Disconnect(ctx)
		} else {
			err = app.RunCtlCommand(ctx, "stop")
		}
	case ActionReload:
		err = app.RunCtlCommand(ctx, "reload")
	case ActionExportLog:
		if pres != nil {
			var path string
			path, err = pres.ExportLog(ctx)
			if err == nil {
				io.Line("лог: " + path)
			}
		} else {
			err = app.RunCtlCommand(ctx, "export-log")
		}
	case ActionUpdateCheck:
		err = app.RunCtlCommand(ctx, "update-check")
	case ActionUpdateInstall:
		err = app.RunCtlCommand(ctx, "update-install")
	case ActionProfiles:
		err = app.ListProfiles(ctx)
	case ActionImport:
		uri, e := io.ReadLine("URI: ")
		if e != nil {
			return 1
		}
		if uri == "" {
			io.Line("пустой URI")
			return 1
		}
		err = app.ImportURI(ctx, uri)
	case ActionChain:
		raw, e := io.ReadLine("ID через запятую (например 1,2,3): ")
		if e != nil {
			return 1
		}
		ids, e := parseIDs(raw)
		if e != nil {
			io.Line(e.Error())
			return 1
		}
		err = app.RunChain(ctx, ids)
	case ActionAdapt:
		err = app.Adapt(ctx)
	case ActionServiceMode:
		err = app.ToggleServiceMode(ctx)
		if err == nil && pres != nil {
			_ = pres.Refresh(ctx)
			c, s := pres.Snapshot()
			printStatusBanner(io, c, s)
		}
	case ActionRouteQuick:
		return RunRouteMenu(ctx, app, io)
	case ActionSettings:
		return RunSettingsMenu(ctx, app, io)
	case ActionGroups:
		return RunGroupsMenu(ctx, app, io)
	case ActionDaemon:
		return RunDaemonControl(ctx, app, io, daemonArgs)
	default:
		return 1
	}
	if err != nil {
		io.Line("→ " + err.Error())
		return 1
	}
	return 0
}

func runPing(ctx context.Context, app *appcore.App, pres *presenter.Presenter, io *console.IO) error {
	if pres == nil {
		return app.RunCtlCommand(ctx, "ping")
	}
	c, _ := pres.Snapshot()
	if len(c.BulkMembers) > 0 && c.Connected {
		io.Line("Пинг всех ног пула…")
		resp, err := pres.BulkPingAll(ctx)
		if err != nil {
			return err
		}
		if resp.Error != "" {
			io.Line("→ " + resp.Error)
		}
		io.Line(fmt.Sprintf("Готово: %d/%d живых", resp.OK, resp.Total))
		c, _ = pres.Snapshot()
		printBulkMembers(io, c)
		return nil
	}
	io.Line("Пинг активного прокси…")
	resp, err := pres.Ping(ctx)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		io.Line("→ " + resp.Error)
	} else if resp.DelayMs > 0 {
		io.Line(fmt.Sprintf("Задержка: %d ms (%s)", resp.DelayMs, resp.ProxyName))
	}
	return nil
}

func parseIDs(raw string) ([]int64, error) {
	var ids []int64
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("неверный id: %q", p)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("укажите хотя бы один id")
	}
	return ids, nil
}
