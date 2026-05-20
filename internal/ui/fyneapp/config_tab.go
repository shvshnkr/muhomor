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
	onBack      func() // set by shell: return to simple mode

	groupSelect   *widget.Select
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
}

func newConfigTab(w fyne.Window, app *appcore.App, ctx context.Context, onBack func()) *configTab {
	c := &configTab{w: w, app: app, ctx: ctx, onBack: onBack}
	c.statusLabel = widget.NewLabel("")
	c.kindLabel = widget.NewLabel("")
	c.groupSelect = widget.NewSelect([]string{}, c.onGroupChanged)
	c.uaLabel = widget.NewLabel("UA: авто")
	c.subEntry = widget.NewEntry()
	c.subEntry.SetPlaceHolder("https://…/sub")
	c.subLabel = widget.NewLabel("URL подписки")

	c.profileList = widget.NewList(
		func() int { return len(c.filtered) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(c.filtered) {
				return
			}
			p := c.filtered[i]
			delay := "—"
			if p.LastDelayMs > 0 {
				delay = fmt.Sprintf("%d ms", p.LastDelayMs)
			}
			o.(*widget.Label).SetText(fmt.Sprintf("%s  (%s)  %s", p.Name, p.Type, delay))
		},
	)
	c.profileList.OnSelected = func(id widget.ListItemID) { c.selectedProf = int(id) }

	backBtn := widget.NewButton("← Простой режим", func() { c.onBack() })
	addGroupBtn := widget.NewButton("Новая группа", c.promptNewGroup)
	delGroupBtn := widget.NewButton("Удалить группу", c.deleteGroup)
	saveGroupBtn := widget.NewButton("Сохранить", c.saveGroup)
	c.refreshSubBtn = widget.NewButton("Обновить подписку", c.refreshSubscription)
	c.addServerBtn = widget.NewButton("Добавить сервер", c.promptAddServer)
	testAllBtn := widget.NewButton("Тест списка", c.testAllDelays)
	delayBtn := widget.NewButton("Задержка", c.testOneDelay)
	delProfBtn := widget.NewButton("Удалить профиль", c.deleteProfile)
	refreshBtn := widget.NewButton("Обновить список", func() { c.load(ctx) })

	c.subBox = container.NewVBox(c.subLabel, c.subEntry)
	toolbar := container.NewHBox(
		addGroupBtn, delGroupBtn, saveGroupBtn, c.refreshSubBtn, c.addServerBtn,
		testAllBtn, delayBtn, delProfBtn, refreshBtn,
	)
	c.content = container.NewBorder(
		container.NewVBox(backBtn, widget.NewLabel("Группы и профили"), c.groupSelect, c.kindLabel,
			c.subBox, c.uaLabel, toolbar, c.statusLabel),
		nil, nil, nil,
		container.NewScroll(c.profileList),
	)
	c.updateKindUI()
	return c
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
			msg = "Демон устарел — остановите и запустите заново (Настройки → демон), затем «Обновить список»"
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
		names := make([]string, len(groups))
		for i, g := range groups {
			names[i] = c.groupLabel(g)
		}
		c.groupSelect.SetOptions(names)
		if c.selectedGID == 0 && len(groups) > 0 {
			c.selectedGID = groups[0].ID
			c.selectedKind = groups[0].Kind
			c.groupSelect.SetSelected(names[0])
			c.subEntry.SetText(groups[0].SubscriptionLink)
			c.updateUALabel(groups[0])
		}
		c.applyGroupFilter()
		c.updateKindUI()
		c.statusLabel.SetText(fmt.Sprintf("Групп: %d, профилей: %d", len(groups), len(profiles)))
	})
}

func (c *configTab) onGroupChanged(name string) {
	for _, g := range c.groups {
		if c.groupLabel(g) == name {
			c.selectedGID = g.ID
			c.selectedKind = g.Kind
			c.subEntry.SetText(g.SubscriptionLink)
			c.updateUALabel(g)
			c.applyGroupFilter()
			c.updateKindUI()
			return
		}
	}
}

func (c *configTab) updateUALabel(g apiclient.Group) {
	mode := g.UserAgentMode
	if mode == "" {
		mode = "авто"
	}
	c.uaLabel.SetText("UA подписки: " + mode + " (подбирается автоматически)")
}

func (c *configTab) updateKindUI() {
	isSub := c.selectedKind == store.GroupKindSubscription
	if isSub {
		c.kindLabel.SetText("Тип: подписка (HTTP fetch + User-Agent)")
		c.subBox.Show()
		c.refreshSubBtn.Show()
		c.addServerBtn.Hide()
	} else {
		c.kindLabel.SetText("Тип: ручная (отдельные серверы)")
		c.subBox.Hide()
		c.refreshSubBtn.Hide()
		c.addServerBtn.Show()
	}
}

func (c *configTab) applyGroupFilter() {
	c.filtered = nil
	for _, p := range c.profiles {
		if c.selectedGID == 0 || p.GroupID == c.selectedGID {
			c.filtered = append(c.filtered, p)
		}
	}
	c.profileList.Refresh()
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
				c.uaLabel.SetText("UA подписки: " + mode + " (сохранён)")
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
	c.statusLabel.SetText("Тест списка…")
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
		dialog.ShowInformation("Профиль", "Выберите профиль", c.w)
		return
	}
	id := c.filtered[c.selectedProf].ID
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
				c.statusLabel.SetText(fmt.Sprintf("Задержка: %d ms", res.DelayMs))
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
