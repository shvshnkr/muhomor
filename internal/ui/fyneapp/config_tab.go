//go:build cgo

package fyneapp

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/muhomor/muhomor/internal/apiclient"
	"github.com/muhomor/muhomor/internal/appcore"
	"github.com/muhomor/muhomor/internal/store"
)

type configTab struct {
	content     fyne.CanvasObject
	w           fyne.Window
	app         *appcore.App
	ctx         context.Context
	onBack      func()

	groupList     *widget.List
	filterEntry   *widget.Entry
	kindLabel     *widget.Label
	uaLabel       *widget.Label
	subEntry      *widget.Entry
	subLabel      *widget.Label
	profileList   *widget.List
	statusLabel   *widget.Label
	refreshSubBtn *widget.Button
	addServerBtn  *widget.Button
	subBox        *fyne.Container

	groups       []apiclient.Group
	profiles     []apiclient.Profile
	filtered     []apiclient.Profile
	selectedGID  int64
	selectedKind string
	selectedProf int
	selectedGroup int
	filterText   string
}

func newConfigTab(w fyne.Window, app *appcore.App, ctx context.Context, onBack func()) *configTab {
	c := &configTab{w: w, app: app, ctx: ctx, onBack: onBack}
	c.statusLabel = widget.NewLabel("")
	c.statusLabel.Wrapping = fyne.TextWrapWord
	c.kindLabel = widget.NewLabel("")
	c.kindLabel.Wrapping = fyne.TextWrapWord
	c.uaLabel = widget.NewLabel("UA: авто")
	c.uaLabel.Wrapping = fyne.TextWrapWord
	c.subEntry = widget.NewEntry()
	c.subEntry.SetPlaceHolder("https://…/sub")
	c.subLabel = widget.NewLabel("URL подписки")
	c.filterEntry = widget.NewEntry()
	c.filterEntry.SetPlaceHolder("Поиск по имени сервера…")
	c.filterEntry.OnChanged = func(s string) {
		c.filterText = strings.TrimSpace(s)
		c.applyGroupFilter()
	}

	c.groupList = widget.NewList(
		func() int { return len(c.groups) },
		func() fyne.CanvasObject {
			l := widget.NewLabel("group")
			l.Truncation = fyne.TextTruncateEllipsis
			return l
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(c.groups) {
				return
			}
			o.(*widget.Label).SetText(c.groupLabel(c.groups[i]))
		},
	)
	c.groupList.OnSelected = func(id widget.ListItemID) {
		c.selectedGroup = int(id)
		if id >= 0 && int(id) < len(c.groups) {
			g := c.groups[id]
			c.selectedGID = g.ID
			c.selectedKind = g.Kind
			c.subEntry.SetText(g.SubscriptionLink)
			c.updateUALabel(g)
			c.applyGroupFilter()
			c.updateKindUI()
		}
	}

	c.profileList = widget.NewList(
		func() int { return len(c.filtered) },
		func() fyne.CanvasObject { return profileListRowTemplate() },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(c.filtered) {
				return
			}
			p := c.filtered[i]
			delay := "—"
			if p.LastDelayMs > 0 {
				delay = fmt.Sprintf("%d ms", p.LastDelayMs)
			}
			fillProfileListRow(o, truncateRunes(p.Name, 72), p.Type, delay)
		},
	)
	c.profileList.OnSelected = func(id widget.ListItemID) { c.selectedProf = int(id) }

	backBtn := backButton("← Простой режим", func() { c.onBack() })
	addGroupBtn := widget.NewButton("Новая группа", c.promptNewGroup)
	delGroupBtn := widget.NewButton("Удалить", c.deleteGroup)
	delGroupBtn.Importance = widget.DangerImportance
	saveGroupBtn := widget.NewButton("Сохранить", c.saveGroup)
	saveGroupBtn.Importance = widget.HighImportance
	c.refreshSubBtn = widget.NewButton("Обновить подписку", c.refreshSubscription)
	c.addServerBtn = widget.NewButton("Добавить сервер", c.promptAddServer)
	testAllBtn := widget.NewButton("Тест списка", c.testAllDelays)
	delayBtn := widget.NewButton("Задержка", c.testOneDelay)
	delProfBtn := widget.NewButton("Удалить профиль", c.deleteProfile)
	delProfBtn.Importance = widget.DangerImportance
	refreshBtn := widget.NewButton("Обновить", func() { c.load(ctx) })

	c.subBox = container.NewVBox(c.subLabel, c.subEntry)
	c.updateKindUI()

	groupScroll := container.NewScroll(c.groupList)
	groupScroll.SetMinSize(fyne.NewSize(sidebarW, 180))
	left := container.NewVBox(
		sectionHeader("Подписки", "Группы и источники"),
		groupScroll,
		c.kindLabel,
		c.subBox,
		c.uaLabel,
		container.NewHBox(addGroupBtn, delGroupBtn, saveGroupBtn, refreshBtn),
	)
	leftCard := surfaceCard(left, themePadding())

	c.refreshSubBtn.Importance = widget.HighImportance
	c.addServerBtn.Importance = widget.HighImportance
	profToolbar := configToolbar(
		[]fyne.CanvasObject{c.refreshSubBtn, c.addServerBtn},
		[]fyne.CanvasObject{testAllBtn, delayBtn},
		[]fyne.CanvasObject{delProfBtn},
	)
	right := container.NewBorder(
		container.NewVBox(
			sectionHeader("Серверы", "Профили выбранной группы"),
			c.filterEntry,
			c.statusLabel,
		),
		profToolbar,
		nil, nil,
		listPanel(c.profileList),
	)
	rightCard := surfaceCard(right, themePadding())

	split := container.NewHSplit(leftCard, rightCard)
	split.Offset = 0.28

	c.content = container.NewPadded(container.NewBorder(backBtn, nil, nil, nil, split))
	return c
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func (c *configTab) groupLabel(g apiclient.Group) string {
	k := g.Kind
	if k == "" {
		k = "?"
	}
	return fmt.Sprintf("%s [%s] (%d)", g.Name, k, g.ProfileCount)
}

func (c *configTab) load(ctx context.Context) {
	if c.app.Groups == nil {
		return
	}
	groups, err := c.app.Groups.ListGroups(ctx)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "404") {
			msg = "Демон устарел — остановите и запустите заново (Настройки → демон), затем «Обновить»"
		}
		fyne.Do(func() { c.statusLabel.SetText("Ошибка: " + msg) })
		return
	}
	profiles, err := c.app.Config.ListProfiles(ctx)
	if err != nil {
		fyne.Do(func() { c.statusLabel.SetText("Ошибка: " + err.Error()) })
		return
	}
	fyne.Do(func() {
		c.groups = groups
		c.profiles = profiles
		c.groupList.Refresh()
		if c.selectedGID == 0 && len(groups) > 0 {
			c.selectedGroup = 0
			c.selectedGID = groups[0].ID
			c.selectedKind = groups[0].Kind
			c.groupList.Select(0)
			c.subEntry.SetText(groups[0].SubscriptionLink)
			c.updateUALabel(groups[0])
		} else {
			for i, g := range groups {
				if g.ID == c.selectedGID {
					c.selectedGroup = i
					c.groupList.Select(i)
					break
				}
			}
		}
		c.applyGroupFilter()
		c.updateKindUI()
		c.statusLabel.SetText(fmt.Sprintf("Групп: %d · в группе: %d · всего профилей: %d",
			len(groups), len(c.filtered), len(profiles)))
	})
}

func (c *configTab) updateUALabel(g apiclient.Group) {
	mode := g.UserAgentMode
	if mode == "" {
		mode = "авто"
	}
	c.uaLabel.SetText("UA: " + mode)
}

func (c *configTab) updateKindUI() {
	isSub := c.selectedKind == store.GroupKindSubscription
	if isSub {
		c.kindLabel.SetText("Тип: подписка")
		c.subBox.Show()
		c.refreshSubBtn.Show()
		c.addServerBtn.Hide()
	} else {
		c.kindLabel.SetText("Тип: ручная")
		c.subBox.Hide()
		c.refreshSubBtn.Hide()
		c.addServerBtn.Show()
	}
}

func (c *configTab) applyGroupFilter() {
	c.filtered = nil
	q := strings.ToLower(c.filterText)
	for _, p := range c.profiles {
		if c.selectedGID != 0 && p.GroupID != c.selectedGID {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(p.Name), q) {
			continue
		}
		c.filtered = append(c.filtered, p)
	}
	c.profileList.Refresh()
	if len(c.filtered) > 0 {
		c.statusLabel.SetText(fmt.Sprintf("Показано серверов: %d", len(c.filtered)))
	} else if c.selectedGID != 0 {
		c.statusLabel.SetText("В группе нет профилей (или ничего не найдено по поиску)")
	}
}

func (c *configTab) promptNewGroup() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Имя")
	kindRadio := widget.NewRadioGroup([]string{"Подписка", "Ручная"}, nil)
	kindRadio.SetSelected("Подписка")
	subEntry := widget.NewEntry()
	subEntry.SetPlaceHolder("URL подписки (для типа Подписка)")
	items := []*widget.FormItem{
		{Text: "Имя", Widget: nameEntry},
		{Text: "Тип", Widget: kindRadio},
		{Text: "URL", Widget: subEntry},
	}
	dialog.ShowForm("Новая группа", "Создать", "Отмена", items, func(ok bool) {
		if !ok || strings.TrimSpace(nameEntry.Text) == "" {
			return
		}
		kind := store.GroupKindManual
		if kindRadio.Selected == "Подписка" {
			kind = store.GroupKindSubscription
		}
		go func() {
			_, err := c.app.Groups.CreateGroup(c.ctx, apiclient.GroupRequest{
				Name: nameEntry.Text, Kind: kind, SubscriptionLink: strings.TrimSpace(subEntry.Text),
			})
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, c.w)
					return
				}
				c.load(c.ctx)
			})
		}()
	}, c.w)
}

func (c *configTab) promptAddServer() {
	if c.selectedGID == 0 || c.selectedKind != store.GroupKindManual {
		dialog.ShowInformation("Группа", "Выберите ручную группу", c.w)
		return
	}
	uriEntry := widget.NewEntry()
	uriEntry.SetPlaceHolder("vless://… или trojan://…")
	dialog.ShowForm("Добавить сервер", "Добавить", "Отмена", []*widget.FormItem{
		{Text: "URI", Widget: uriEntry},
	}, func(ok bool) {
		if !ok || strings.TrimSpace(uriEntry.Text) == "" {
			return
		}
		go func() {
			_, err := c.app.Groups.AddGroupServer(c.ctx, c.selectedGID, strings.TrimSpace(uriEntry.Text))
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, c.w)
					return
				}
				c.statusLabel.SetText("Сервер добавлен")
				c.load(c.ctx)
			})
		}()
	}, c.w)
}

func (c *configTab) refreshSubscription() {
	if c.selectedGID == 0 || c.selectedKind != store.GroupKindSubscription {
		return
	}
	_ = c.saveGroupSync()
	c.statusLabel.SetText("Загрузка подписки…")
	go func() {
		out, err := c.app.Groups.RefreshGroup(c.ctx, c.selectedGID)
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, c.w)
				return
			}
			if mode, ok := out["user_agent_mode"].(string); ok && mode != "" {
				c.uaLabel.SetText("UA: " + mode)
			}
			c.statusLabel.SetText(fmt.Sprintf("Подписка обновлена: %v", out))
			c.load(c.ctx)
		})
	}()
}

func (c *configTab) saveGroupSync() error {
	if c.selectedGID == 0 {
		return nil
	}
	return c.app.Groups.UpdateGroup(c.ctx, c.selectedGID, apiclient.GroupRequest{
		SubscriptionLink: c.subEntry.Text,
	})
}

func (c *configTab) deleteGroup() {
	if c.selectedGID == 0 {
		return
	}
	dialog.ShowConfirm("Удалить группу", "Удалить группу и все её профили?", func(ok bool) {
		if !ok {
			return
		}
		go func() {
			err := c.app.Groups.DeleteGroup(c.ctx, c.selectedGID)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, c.w)
					return
				}
				c.selectedGID = 0
				c.load(c.ctx)
			})
		}()
	}, c.w)
}

func (c *configTab) saveGroup() {
	if c.selectedGID == 0 {
		return
	}
	go func() {
		err := c.saveGroupSync()
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, c.w)
				return
			}
			c.statusLabel.SetText("Сохранено")
		})
	}()
}

func (c *configTab) testAllDelays() {
	if c.selectedGID == 0 {
		return
	}
	c.statusLabel.SetText("Тест списка… (может занять минуту)")
	go func() {
		out, err := c.app.Groups.TestGroupDelays(c.ctx, c.selectedGID)
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, c.w)
				return
			}
			c.statusLabel.SetText(fmt.Sprintf("Тест: %v", out))
			c.load(c.ctx)
		})
	}()
}

func (c *configTab) testOneDelay() {
	if c.selectedProf < 0 || c.selectedProf >= len(c.filtered) {
		dialog.ShowInformation("Профиль", "Выберите профиль в списке справа", c.w)
		return
	}
	id := c.filtered[c.selectedProf].ID
	name := c.filtered[c.selectedProf].Name
	c.statusLabel.SetText("Тест: " + truncateRunes(name, 40) + "…")
	go func() {
		res, err := c.app.Groups.TestProfileDelay(c.ctx, id)
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, c.w)
				return
			}
			if res.Error != "" {
				c.statusLabel.SetText(res.Error)
			} else {
				c.statusLabel.SetText(fmt.Sprintf("%s — %d ms", truncateRunes(name, 48), res.DelayMs))
			}
			c.load(c.ctx)
		})
	}()
}

func (c *configTab) deleteProfile() {
	if c.selectedProf < 0 || c.selectedProf >= len(c.filtered) {
		return
	}
	p := c.filtered[c.selectedProf]
	if p.WLBuiltinPool {
		dialog.ShowInformation("Встроенный", "Нельзя удалить", c.w)
		return
	}
	dialog.ShowConfirm("Удалить", "Удалить "+p.Name+"?", func(ok bool) {
		if !ok {
			return
		}
		go func() {
			err := c.app.Groups.DeleteProfile(c.ctx, p.ID)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, c.w)
					return
				}
				c.load(c.ctx)
			})
		}()
	}, c.w)
}
