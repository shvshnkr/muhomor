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

## Запуск

```bash
# GUI поднимет демон сам, если не запущен
./muhomor-gui -d ~/.local/share/muhomor --service-mode proxy --mixed-port 2181
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
