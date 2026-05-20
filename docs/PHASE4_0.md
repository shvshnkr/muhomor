# Phase 4.0 — REST API + remote appcore

Реализовано для desktop GUI (Phase 4.1) и Android (позже).

## Daemon API

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/v1/service/status` | JSON `api.ServiceStatus` (lowercase keys) |
| GET | `/v1/events` | SSE: `status`, `adapt`, `settings` |
| GET/PUT | `/v1/settings` | Настройки store |
| GET | `/v1/profiles` | `{"profiles":[...]}` |
| POST | `/v1/profiles/import` | JSON `{uri}` / `{lines}` или multipart `file` |
| PUT | `/v1/profiles/{id}/enabled` | `{"enabled":true}` |
| POST | `/v1/profiles/{id}/connect` | Подключить профиль |
| POST | `/v1/service/ping` | url-test + `desktop-control-ping.txt` |

Существующие пути без изменений метода: start, stop, reload, chain, adapt, logs, update.

## Клиент

- `internal/apiclient` — все методы + `StreamEvents`
- `internal/appcore/RemoteConfig` — ConfigRepository через HTTP
- `internal/ui/pseudogui` — **без** прямого SQLite

## Пакеты

- `internal/api` — DTO
- `internal/profiles` — импорт URI (daemon + переиспользование)
- `internal/controller/events.go` — EventHub SSE

## Пример

```bash
./muhomor --daemon -d ~/.local/share/muhomor
curl --unix-socket ~/.local/share/muhomor/run/muhomor.sock http://localhost/v1/profiles
curl --unix-socket ... http://localhost/v1/settings
```

## Дальше (4.1)

`cmd/muhomor-gui` на Fyne, Simple screen + tray.
