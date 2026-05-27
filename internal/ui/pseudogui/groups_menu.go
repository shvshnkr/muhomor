package pseudogui

import (
	"context"
	"fmt"
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
		labels := make([]string, 0, len(groups)+2)
		ids := make([]int64, 0, len(groups))
		for _, g := range groups {
			kind := g.Kind
			if kind == "" {
				kind = "?"
			}
			labels = append(labels, fmt.Sprintf("[%d] %s  %s  profiles=%d", g.ID, g.Name, kind, g.ProfileCount))
			ids = append(ids, g.ID)
		}
		labels = append(labels, "[n] Новая группа", "[q] Назад")
		idx, ok := RunListPicker(ctx, io, "Группы", labels, 0)
		if !ok {
			return 0
		}
		if idx == len(labels)-1 {
			return 0
		}
		if idx == len(labels)-2 {
			if code := promptNewGroup(ctx, app, io); code != 0 {
				return code
			}
			continue
		}
		if code := runGroupActions(ctx, app, io, ids[idx]); code != 0 {
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
	typeOpts := []string{"[m] Ручная (manual)", "[s] Подписка (subscription)"}
	tidx, ok := RunListPicker(ctx, io, "Тип группы", typeOpts, 0)
	if !ok {
		return 0
	}
	kind := store.GroupKindManual
	link := ""
	if tidx == 1 {
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
	actionLabels := []string{
		"[r] Обновить подписку",
		"[a] Добавить сервер (URI)",
		"[t] Тест списка",
		"[d] Задержка профиля",
		"[x] Удалить профиль",
		"[u] URL подписки",
		"[q] Назад",
	}
	for {
		mode := g.UserAgentMode
		if mode == "" {
			mode = "авто"
		}
		title := fmt.Sprintf("Группа %d %q (%s) UA:%s", g.ID, g.Name, g.Kind, mode)
		idx, ok := RunListPicker(ctx, io, title, actionLabels, 0)
		if !ok || idx == len(actionLabels)-1 {
			return 0
		}
		switch idx {
		case 0:
			if g.Kind != store.GroupKindSubscription {
				io.Line("только для групп-подписок")
				continue
			}
			out, err := app.Groups.RefreshGroup(ctx, groupID)
			if err != nil {
				io.Line(err.Error())
			} else {
				io.Line(fmt.Sprintf("refresh: %v", out))
			}
		case 1:
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
		case 2:
			out, err := app.Groups.TestGroupDelays(ctx, groupID)
			if err != nil {
				io.Line(err.Error())
			} else {
				io.Line(fmt.Sprintf("тест: %v", out))
			}
		case 3:
			pid, ok := pickGroupProfile(ctx, app, io, groupID)
			if !ok {
				continue
			}
			res, err := app.Groups.TestProfileDelay(ctx, pid)
			if err != nil {
				io.Line(err.Error())
			} else {
				io.Line(fmt.Sprintf("delay: %+v", res))
			}
		case 4:
			pid, ok := pickGroupProfile(ctx, app, io, groupID)
			if !ok {
				continue
			}
			if err := app.Groups.DeleteProfile(ctx, pid); err != nil {
				io.Line(err.Error())
			} else {
				io.Line("удалён")
			}
		case 5:
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
		}
	}
}

func pickGroupProfile(ctx context.Context, app *appcore.App, io *console.IO, groupID int64) (int64, bool) {
	all, err := app.Config.ListProfiles(ctx)
	if err != nil {
		io.Line(err.Error())
		return 0, false
	}
	var labels []string
	var ids []int64
	for _, p := range all {
		if p.GroupID != groupID {
			continue
		}
		name := p.Name
		if name == "" {
			name = "?"
		}
		labels = append(labels, fmt.Sprintf("[%d] %s", p.ID, truncate(name, 36)))
		ids = append(ids, p.ID)
	}
	if len(ids) == 0 {
		io.Line("в группе нет профилей")
		return 0, false
	}
	labels = append(labels, "[q] Отмена")
	idx, ok := RunListPicker(ctx, io, "Профиль", labels, 0)
	if !ok || idx == len(labels)-1 {
		return 0, false
	}
	return ids[idx], true
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// RunDaemonControl start/stop daemon process (extended settings).
func RunDaemonControl(ctx context.Context, app *appcore.App, io *console.IO, daemonArgs []string) int {
	opts := []string{
		"[1] Запустить демон",
		"[2] Остановить демон",
		"[q] Назад",
	}
	idx, ok := RunListPicker(ctx, io, "Демон", opts, 0)
	if !ok || idx == 2 {
		return 0
	}
	switch idx {
	case 0:
		if err := platform.EnsureDaemon(ctx, app.Layout, daemonArgs); err != nil {
			io.Line(err.Error())
			return 1
		}
		io.Line("демон запущен")
	case 1:
		if err := platform.StopDaemon(ctx, app.Layout); err != nil {
			io.Line(err.Error())
			return 1
		}
		io.Line("демон остановлен")
	}
	return 0
}
