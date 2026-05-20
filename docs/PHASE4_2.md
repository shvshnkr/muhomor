# Phase 4.2 — Full GUI (muhomor-gui)

**Статус:** реализовано (Windows/Linux, Fyne + CGO)

## Вкладки

| Вкладка | Содержание |
|---------|------------|
| **Простой** | Connect/Disconnect, activity, proxy/VPN, задержка текущего профиля, экспорт лога, «Полный режим →» |
| **Конфигурация** | Группы (подписки и ручные сборники), User-Agent, импорт URI/подписки, тест всего списка + автосортировка, задержка/удаление профиля |
| **Маршрут** | Пресеты `route_quick_profile` 0/1/2 (как Dahusim) |
| **Настройки** | mixed-port, service mode, reload, запуск демона |

Навигация «← Простой режим» на полных вкладках.

## REST API (демон)

| Метод | Путь |
|-------|------|
| GET/POST | `/v1/groups` |
| PUT/DELETE | `/v1/groups/{id}` |
| POST | `/v1/groups/{id}/import` |
| POST | `/v1/groups/{id}/test-all` |
| DELETE | `/v1/profiles/{id}` |
| POST | `/v1/profiles/{id}/delay-test` |

User-Agent подписки: **автоматически** (как Dahusim `SubscriptionFetchProfile`):
- GitHub raw → `husi/…` (default), затем happ, browser
- остальные URL → **happ/2.9.0** (default), затем browser
- успешный UA сохраняется в KV `group:{id}:user_agent` (пользователь не редактирует)

## Сборка

```powershell
go build -tags cgo -o muhomor-gui.exe ./cmd/muhomor-gui
.\scripts\start-gui-windows.ps1
```

## Типы групп

| `kind` | Назначение |
|--------|------------|
| `subscription` | URL подписки + User-Agent, кнопка «Обновить подписку», авто-fetch ~45 мин (как Dahusim H29) |
| `manual` | Только отдельные URI («Добавить сервер»), без HTTP sub |

## Pseudo-GUI

Те же возможности: `[g]` группы, `[d]` демон start/stop.

## API

| POST | `/v1/groups/{id}/refresh` | fetch подписки |
| POST | `/v1/groups/{id}/servers` | один URI в ручную группу |
| POST | `/v1/daemon/shutdown` | остановка демона |

## Ограничения

- Встроенная WL-группа не удаляется; встроенные профили не удаляются.
