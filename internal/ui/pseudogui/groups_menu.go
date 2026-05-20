package pseudogui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/platform"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/ui/console"
)

// RunGroupsMenu interactive group management (parity with Fyne Configuration tab).
func RunGroupsMenu(ctx context.Context, app *appcore.App, io *console.IO) int {
	if app.Groups == nil {
		io.Line("groups API unavailable")
		return 1
	}
	for {
		groups, err := app.Groups.ListGroups(ctx)
		if err != nil {
			io.Line(err.Error())
			return 1
		}
		io.Line("")
		io.Line("--- Группы ---")
		for _, g := range groups {
			kind := g.Kind
			if kind == "" {
				kind = "?"
			}
			io.Line(fmt.Sprintf("  [%d] %s  kind=%s  profiles=%d  sub=%q", g.ID, g.Name, kind, g.ProfileCount, truncate(g.SubscriptionLink, 40)))
		}
		io.Line("  [n] новая группа  [q] назад")
		choice, err := io.ReadLine("ID группы или команда: ")
		if err != nil {
			return 0
		}
		choice = strings.TrimSpace(choice)
		if choice == "q" || choice == "" {
			return 0
		}
		if choice == "n" {
			if code := promptNewGroup(ctx, app, io); code != 0 {
				return code
			}
			continue
		}
		id, err := strconv.ParseInt(choice, 10, 64)
		if err != nil {
			io.Line("нужен числовой id")
			continue
		}
		if code := runGroupActions(ctx, app, io, id); code != 0 {
			return code
		}
	}
}

func promptNewGroup(ctx context.Context, app *appcore.App, io *console.IO) int {
	name, _ := io.ReadLine("Имя группы: ")
	name = strings.TrimSpace(name)
	if name == "" {
		return 1
	}
	kindRaw, _ := io.ReadLine("Тип: [s] подписка / [m] ручная: ")
	kind := store.GroupKindManual
	link := ""
	if strings.TrimSpace(strings.ToLower(kindRaw)) == "s" {
		kind = store.GroupKindSubscription
		link, _ = io.ReadLine("URL подписки: ")
	}
	_, err := app.Groups.CreateGroup(ctx, apiclient.GroupRequest{
		Name: name, Kind: kind, SubscriptionLink: strings.TrimSpace(link),
	})
	if err != nil {
		io.Line(err.Error())
		return 1
	}
	io.Line("группа создана")
	return 0
}

func runGroupActions(ctx context.Context, app *appcore.App, io *console.IO, groupID int64) int {
	groups, err := app.Groups.ListGroups(ctx)
	if err != nil {
		io.Line(err.Error())
		return 1
	}
	var g *apiclient.Group
	for i := range groups {
		if groups[i].ID == groupID {
			g = &groups[i]
			break
		}
	}
	if g == nil {
		io.Line("группа не найдена")
		return 1
	}
	for {
		mode := g.UserAgentMode
		if mode == "" {
			mode = "авто"
		}
		io.Line(fmt.Sprintf("Группа %d %q (%s)  UA:%s", g.ID, g.Name, g.Kind, mode))
		io.Line("  [r] обновить подписку  [a] добавить сервер  [t] тест списка  [d] задержка  [x] удалить  [u] URL  [q] назад")
		a, _ := io.ReadLine("действие: ")
		switch strings.TrimSpace(strings.ToLower(a)) {
		case "q", "":
			return 0
		case "r":
			if g.Kind != store.GroupKindSubscription {
				io.Line("только для групп-подписок")
				continue
			}
			out, err := app.Groups.RefreshGroup(ctx, groupID)
			if err != nil {
				io.Line(err.Error())
			} else {
				if m, ok := out["user_agent_mode"].(string); ok {
					io.Line(fmt.Sprintf("refresh OK, UA=%s, %v", m, out))
				} else {
					io.Line(fmt.Sprintf("refresh: %v", out))
				}
			}
		case "a":
			if g.Kind != store.GroupKindManual {
				io.Line("только для ручных групп")
				continue
			}
			uri, _ := io.ReadLine("URI (vless://…): ")
			if uri == "" {
				continue
			}
			res, err := app.Groups.AddGroupServer(ctx, groupID, strings.TrimSpace(uri))
			if err != nil {
				io.Line(err.Error())
			} else {
				io.Line(fmt.Sprintf("добавлено: %d", len(res)))
			}
		case "t":
			out, err := app.Groups.TestGroupDelays(ctx, groupID)
			if err != nil {
				io.Line(err.Error())
			} else {
				io.Line(fmt.Sprintf("тест: %v", out))
			}
		case "d":
			raw, _ := io.ReadLine("id профиля: ")
			pid, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
			if err != nil {
				continue
			}
			res, err := app.Groups.TestProfileDelay(ctx, pid)
			if err != nil {
				io.Line(err.Error())
			} else {
				io.Line(fmt.Sprintf("delay: %+v", res))
			}
		case "x":
			raw, _ := io.ReadLine("id профиля: ")
			pid, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
			if err != nil {
				continue
			}
			if err := app.Groups.DeleteProfile(ctx, pid); err != nil {
				io.Line(err.Error())
			} else {
				io.Line("удалён")
			}
		case "u":
			link, _ := io.ReadLine(fmt.Sprintf("URL подписки [%s]: ", g.SubscriptionLink))
			req := apiclient.GroupRequest{}
			if strings.TrimSpace(link) != "" {
				req.SubscriptionLink = strings.TrimSpace(link)
			}
			if err := app.Groups.UpdateGroup(ctx, groupID, req); err != nil {
				io.Line(err.Error())
			} else {
				io.Line("сохранено")
			}
			groups, _ = app.Groups.ListGroups(ctx)
			for i := range groups {
				if groups[i].ID == groupID {
					g = &groups[i]
				}
			}
		default:
			io.Line("неизвестно")
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// RunDaemonControl start/stop daemon process (extended settings).
func RunDaemonControl(ctx context.Context, app *appcore.App, io *console.IO, daemonArgs []string) int {
	io.Line("[1] Запустить демон  [2] Остановить демон  [q] назад")
	c, _ := io.ReadLine("выбор: ")
	switch strings.TrimSpace(c) {
	case "1":
		if err := platform.EnsureDaemon(ctx, app.Layout, daemonArgs); err != nil {
			io.Line(err.Error())
			return 1
		}
		io.Line("демон запущен")
	case "2":
		if err := platform.StopDaemon(ctx, app.Layout); err != nil {
			io.Line(err.Error())
			return 1
		}
		io.Line("демон остановлен")
	}
	return 0
}
