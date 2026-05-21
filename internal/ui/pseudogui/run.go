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

// Run interactive menu until quit. Returns exit code.
func Run(ctx context.Context, cfg Config) int {
	app := cfg.App
	io := cfg.IO
	pres := cfg.Pres
	if io == nil {
		io = console.StdIO()
	}

	io.Line("")
	io.Line("muhomor pseudo-GUI — simple mode (selector + post-connect test)")
	io.Line("Прокси: curl -x http://127.0.0.1:<mixed-port> …  (auth на localhost не нужен)")

	for {
		printMenu(io)
		choice, err := io.ReadLine("Выбор: ")
		if err != nil {
			return 0
		}
		action, ok := ParseChoice(choice)
		if !ok {
			io.Line("Неизвестная команда")
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
	io.Line("--- Сервис (simple mode) ---")
	io.Line("[1] Статус")
	io.Line("[2] Ping")
	io.Line("[3] Подключить / отключить")
	io.Line("[4] Остановить")
	io.Line("[5] Перезагрузить")
	io.Line("[6] Экспорт лога")
	io.Line("[7] Проверить обновление")
	io.Line("[8] Установить обновление")
	io.Line("--- Профили ---")
	io.Line("[p] Список профилей")
	io.Line("[i] Импорт URI")
	io.Line("[h] Цепочка relay (chain)")
	io.Line("--- Сеть / настройки ---")
	io.Line("[a] Адаптация сети (handoff)")
	io.Line("[m] Режим proxy / vpn")
	io.Line("[r] Быстрый маршрут (0–3, см. режим WG)")
	io.Line("[s] Настройки (показать / mixed port / multipath)")
	io.Line("[g] Группы (подписки / ручные)")
	io.Line("[d] Демон: запуск / остановка")
	io.Line("[q] Выход")
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
		err = app.RunCtlCommand(ctx, "ping")
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
			io.Line("Подключение (подписки → TCP/URL тест → post-connect)…")
			err = pres.Connect(ctx)
		}
		if err == nil {
			c, s := pres.Snapshot()
			printStatus(io, c, s)
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
			printStatus(io, c, s)
		}
	case ActionRouteQuick:
		raw, e := io.ReadLine("0=manual 1=ru_direct 2=ru_blocked_ai 3=wg_over_wl_tunnel: ")
		if e != nil {
			return 1
		}
		v, e := strconv.Atoi(strings.TrimSpace(raw))
		if e != nil || v < 0 || v > 3 {
			io.Line("нужно 0, 1, 2 или 3")
			return 1
		}
		err = app.SetRouteQuick(ctx, v)
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
