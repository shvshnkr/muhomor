# Phase 4.1 — Fyne Simple UI + tray

## Сборка

```bash
go build -o muhomor ./cmd/muhomor
go build -o muhomor-gui ./cmd/muhomor-gui   # требует CGO (gcc на Windows)
```

Windows (MSYS2):

```powershell
# один раз: pacman -S mingw-w64-ucrt-x86_64-gcc
.\scripts\fetch-mihomo-windows.ps1   # bin\mihomo.exe
.\scripts\setup-windows-dev.ps1      # PATH + MUHOMOR_MIHOMO_BIN + CGO
go build -o muhomor-gui.exe ./cmd/muhomor-gui
```

gcc: `C:\msys64\ucrt64\bin` (добавьте в системный PATH при желании).

## Режимы: Proxy и VPN (TUN)

| Режим | `service_mode` | Что делает |
|-------|----------------|------------|
| **Proxy** | `proxy` | mixed-port (напр. 2181), браузер/curl `-x http://127.0.0.1:PORT` |
| **VPN** | `vpn` | TUN в YAML mihomo (`tun_enable`), системный туннель |

В GUI: кнопки **Proxy** / **VPN** и пункты tray — меняют настройки и делают reload.

**Connect** = simple mode: **selector** перебирает все включённые профили (bootstrap + подписки + WL pool), не один случайный URI. Открытые энтузиастские сервера могут отвалиться — сработает fallback/cooldown (Phase 2).

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

- Подключить / Отключить (simple mode)
- Статус, профиль, порт, режим proxy/vpn
- Tray: показать, start/stop, proxy/vpn, выход
- Закрытие окна → свернуть в tray

## Отладка Windows

См. `scripts/debug-windows.ps1`
