# Pseudo-GUI (терминальное меню)

Интерактивное меню как в Dahusim (`--pseudo-gui`), плюс профили/импорт/chain для muhomor.

## Запуск

```bash
# Меню само поднимет демон, если не запущен (как muhomor-gui)
./muhomor --pseudo-gui -d ~/.local/share/muhomor --service-mode proxy --mixed-port 2181
```

Или вручную:

```bash
./muhomor --daemon -d ~/.local/share/muhomor --service-mode proxy --mixed-port 2181
./muhomor --pseudo-gui -d ~/.local/share/muhomor --mixed-port 2181
```

Windows: HTTP к демону на `127.0.0.1:8751`. Прокси на `mixed-port` **без логина** на localhost.

**[3] Подключить** — тот же simple mode, что в Fyne: selector, URL-тест, post-connect проверка, fallback. В терминале печатаются строки активности (`» Обновление подписок…`, `» TCP тест …`). В статусе **[1]** — probe/multipath из API (если включены).

**[s] Настройки** — подменю как вкладка «Настройки» в GUI: mixed port, proxy/vpn, multipath вкл/выкл, preset `low|normal|high`, WL emergency-only.

## Меню

| Клавиша | Действие | Dahusim |
|---------|----------|---------|
| 1–8 | status, ping, connect/disconnect, stop, reload, export-log, update-check, update-install | да |
| p | список профилей | — |
| i | импорт URI | — |
| h | chain relay | — |
| a | adapt (handoff) | — |
| m | proxy ↔ vpn (store) | — |
| r | route quick 0/1/2/3 | Fyne «Маршрут» |
| s | настройки: показать, mixed port, proxy/vpn, **multipath** | Fyne «Настройки» |
| g | группы: подписка / ручная, refresh, добавить сервер | Fyne «Конфигурация» |
| d | демон: запуск / **остановка процесса** | Fyne «Настройки» |
| q | выход (демон работает) | да |

## Архитектура (Android-ready)

```
cmd/muhomor --pseudo-gui
    → ui/pseudogui/entry.go
    → ui/presenter         # тот же слой, что Fyne (SSE + poll при connect)
    → appcore.App          → apiclient → daemon HTTP
    → ui/console.IO        # stdin/stdout
```

На Android позже: `appcore` + `ServiceControl` через binder/localhost, UI = Compose (не `ui/console`).

## Пакеты

| Пакет | Назначение |
|-------|------------|
| `internal/apiclient` | HTTP к демону |
| `internal/appcore` | действия меню, интерфейсы |
| `internal/ui/pseudogui` | цикл меню |
| `internal/ui/presenter` | connect/disconnect, статус, activity |
| `internal/ui/console` | stdin/stdout, Windows VT |
