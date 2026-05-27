# Phase 4.1 — Wails desktop GUI + tray

## Сборка

```bash
go build -o muhomor ./cmd/muhomor
```

**GUI (Wails):** Node 18+, [Wails CLI](https://wails.io/docs/gettingstarted/installation).

```powershell
# Windows
powershell -File scripts/build-gui-wails.ps1
# или: wails build -platform windows/amd64 -o muhomor-gui.exe
```

```bash
# Linux (нужны dev-пакеты webkit2gtk-4.1, CGO_ENABLED=1)
cd frontend && npm ci && npm run build && cd ..
wails build -platform linux/amd64 -o muhomor-gui
```

Windows: **WebView2 Evergreen** runtime (обычно уже есть на Win10/11).

Linux kit GUI: `webkit2gtk-4.1` (+ dev headers для сборки). См. `kit/linux/README.txt`.

Legacy Fyne (deprecated): `go build -tags fyne -tags cgo …` → см. `internal/ui/fyneapp/DEPRECATED.md`.

## Режимы: Proxy и VPN (TUN)

| Режим | `service_mode` | Что делает |
|-------|----------------|------------|
| **Proxy** | `proxy` | mixed-port (напр. 2181), браузер/curl `-x http://127.0.0.1:PORT` |
| **VPN** | `vpn` | TUN в YAML mihomo (`tun_enable`), системный туннель |

В GUI: кнопки **Proxy** / **VPN** (Simple через tray/настройки) — меняют настройки и делают reload.

**Connect** = simple mode: **selector** перебирает включённые профили (bootstrap + подписки + WL pool), не один случайный URI.

## Запуск

```bash
# GUI поднимет демон сам, если не запущен
./muhomor-gui -d ~/.local/share/muhomor --service-mode proxy --mixed-port 2181
```

Windows:

```powershell
.\scripts\start-gui-windows.ps1
```

Или вручную:

```bash
./muhomor --daemon -d DATA --service-mode proxy --mixed-port 2181
./muhomor-gui -d DATA --mixed-port 2181
```

## Экран Simple

- Подключить / Отключить / Отменить (simple mode)
- Статус, activity, профиль/прокси, probe/standby
- Ping (+ bulk pool)
- Экспорт лога, «Расширенный режим»
- Tray: показать, connect/stop, выход
- Закрытие окна (X) → свернуть в tray

Цели UX: [`UI_PRODUCT_BRIEF.md`](UI_PRODUCT_BRIEF.md).

## Dev

```powershell
cd frontend; npm install; npm run build
wails dev
```

## Отладка Windows

См. `scripts/debug-windows.ps1`
