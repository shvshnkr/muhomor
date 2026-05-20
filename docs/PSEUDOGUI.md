# Pseudo-GUI (терминальное меню)

Интерактивное меню как в Dahusim (`--pseudo-gui`), плюс профили/импорт/chain для muhomor.

## Запуск

```bash
# Терминал 1 — демон
./muhomor --daemon -d ~/.local/share/muhomor

# Терминал 2 — меню
./muhomor --pseudo-gui -d ~/.local/share/muhomor
```

Windows: тот же флаг; сокет Unix недоступен → HTTP на `127.0.0.1:8751`.

## Меню

| Клавиша | Действие | Dahusim |
|---------|----------|---------|
| 1–8 | status, ping, start, stop, reload, export-log, update-check, update-install | да |
| p | список профилей | — |
| i | импорт URI | — |
| h | chain relay | — |
| a | adapt (handoff) | — |
| m | proxy ↔ vpn (store) | — |
| r | route quick 0/1/2 | — |
| s | настройки | — |
| q | выход (демон работает) | да |

## Архитектура (Android-ready)

```
cmd/muhomor --pseudo-gui
    → ui/pseudogui/entry.go
    → appcore.App          # бизнес-логика, без терминала
        ServiceControl     → apiclient → daemon HTTP
        ConfigRepository   → store (SQLite)
    → ui/console.IO        # только desktop TUI
```

На Android позже: `appcore` + `ServiceControl` через binder/localhost, UI = Compose (не `ui/console`).

## Пакеты

| Пакет | Назначение |
|-------|------------|
| `internal/apiclient` | HTTP к демону |
| `internal/appcore` | действия меню, интерфейсы |
| `internal/ui/pseudogui` | цикл меню |
| `internal/ui/console` | stdin/stdout, Windows VT |
