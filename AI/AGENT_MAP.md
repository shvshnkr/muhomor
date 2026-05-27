# Карта для агента (логи, data, kit vs desktop)

**Читать в начале сессии** вместе с `AI/workflowlog.md` (секция **Current focus**). Команды PowerShell — в `AI/CURSOR_SHELL_ALLOWLIST.md`.

---

## 1. Где лежит `data/` (два режима)

| Режим | Как определяется | `DataDir` (Windows) | Mixed port |
|-------|------------------|---------------------|------------|
| **Portable kit** | exe рядом с `.muhomor-portable` или `VERSION.txt` (`muhomor-kit`) + `bin/mihomo*` | `{kit-root}/data` | **2181** |
| **Desktop (dev)** | `go run` / exe **без** маркеров kit | `%LOCALAPPDATA%\muhomor` | **7890** (если не `-mixed-port`) |
| **Явный override** | `-dir PATH` или `MUHOMOR_DATA_DIR` | как задано | из настроек / флага |

Код: `internal/paths/portable.go` (`ResolveDataDir`, `PortableKitRoot`), `DefaultMixedPort`.

**Типовые корни (этой машины):**

| | Путь |
|---|------|
| Kit после pack | `dist/muhomor-kit/` → data = `dist/muhomor-kit/data/` |
| Desktop | `%LOCALAPPDATA%\muhomor` |
| Linux kit | `dist/muhomor-kit-linux-amd64/data/` |

**Kit root (маркеры):** `muhomor.exe` + `bin/mihomo.exe`, `.muhomor-portable`, `config/subscriptions.txt`, `data/muhomor.db`.

**Не путать:** логи kit **только** под `{kit}/data/`, не в корне kit (`cache/` в корне — legacy, pack удаляет).

---

## 2. Дерево `{DataDir}/`

```
{DataDir}/
  muhomor.db              # SQLite — только демон; UI не открывать
  gui.lock                # второй muhomor-gui (PID внутри)
  cache/
    daemon-debug.err.log  # главный лог демона (JSON slog)
    activity.log          # короткие строки UI/connect («Подключение…»)
    simple-mode.log       # снимки state= при export
    desktop-control-status.txt   # ctl: connected, profile
    desktop-control-ping.txt     # последний url-test
    desktop-control-export.txt   # указатели на логи
    daemon-debug.log        # пустой placeholder; legacy debug-windows stdout
  run/
    config.yaml           # сгенерированный mihomo config (secret для API)
    mihomo/
      mihomo-subprocess.log   # stdout/stderr child mihomo
      geoip.metadb            # ~9 MB; kit pack кладёт сюда
      Country.mmdb            # альтернатива geo
    muhomor.sock            # Unix (Linux); Windows → TCP :8751
```

`Layout`: `internal/paths/paths.go` — `CacheDir = {DataDir}/cache`, `RuntimeDir = {DataDir}/run` (или `XDG_RUNTIME_DIR/muhomor`).

---

## 3. Таблица логов (что открывать)

| Файл | Формат | Кто пишет | Когда смотреть |
|------|--------|-----------|----------------|
| `cache/daemon-debug.err.log` | JSON lines (`slog`) | `muhomor --daemon`; stderr при `EnsureDaemon` | connect, selector, fallback, `event=H*` |
| `run/mihomo/mihomo-subprocess.log` | mihomo text | subprocess | geo MMDB, mixed listen, dial timeouts, 502 |
| `cache/activity.log` | `timestamp text` | `Runtime.setActivity` | что видел пользователь в simple mode |
| `cache/simple-mode.log` | `timestamp state=…` | `ExportSimpleLog` | история connect/disconnect |
| `cache/desktop-control-status.txt` | key=value | `WriteStatusFile` | быстрый `connected=` без API |
| `cache/daemon-debug.log` | — | почти не используется | не путать с `.err.log` |

**События в `daemon-debug.err.log` (grep):**

| `event` | Смысл |
|---------|--------|
| `H30` | service/chain connected |
| `H3` | post-connect url-test |
| `H4`, `H4-batch`, `H4-picker` | очередь / pretest / picker |
| `H22`, `H22-retry` | все пробы мертвы / retry WL |
| `BL-exit-filter` | BL-over-proxy filter |
| `H34`, `H34-dial` | health / dial burst |
| `BULK-*` | multipath bulk |
| `H29-*` | scheduler / subs refresh |
| `geodata` | geo update |

**Mihomo subprocess — типовые строки:** `Load MMDB`, `Mixed(http+socks) proxy listening at`, `dial`, `timeout`.

---

## 4. Порты и API (localhost)

| Адрес | Назначение |
|-------|------------|
| `127.0.0.1:8751` | muhomor daemon REST + SSE (`GET /v1/events`) |
| `127.0.0.1:2181` (kit) / `:7890` (desktop) | mixed HTTP/SOCKS proxy |
| `127.0.0.6:9090` | mihomo external-controller (`/version`, `/proxies`, `/delay`) |
| `127.0.0.1:8760` | внутренний picker batch (не для ручного curl) |

Константы: `internal/paths/loopback.go`. WSL: `MUHOMOR_MIHOMO_CONTROLLER_HOST=127.0.0.1` — `docs/WSL.md`.

**Быстрый статус (kit):**

```powershell
$kit = "c:\Users\user\muhomor\dist\muhomor-kit"
Invoke-RestMethod "http://127.0.0.1:8751/v1/service/status" -TimeoutSec 10
Get-Content "$kit\data\cache\desktop-control-status.txt"
```

**Mihomo `/traffic`:** только `curl.exe -m 2`, не `Invoke-WebRequest` (стрим зависает).

**Secret (быстрый путь):**

1. Kit: `dist/muhomor-kit/data/run/config.yaml` → поле `secret:`
2. Desktop/dev: `%LOCALAPPDATA%/muhomor/run/config.yaml` → поле `secret:`
3. Использовать только для локального `Authorization: Bearer ...` к `127.0.0.6:9090`.
4. Если в этих путях нет секрета: сначала уточнить `DataDir`; при необходимости и по согласованию можно делать расширенный поиск, если это ускоряет задачу.

**Рабочий приоритет агента:** ускорять и оптимизировать выполнение задач; экономия токенов считается частью оптимизации (а не самоцелью). Для существенных развилок — короткое сравнение плюсов/минусов и подтверждение выбора.

---

## 5. Чеклист «нет коннекта» (не пересканировать репо)

1. `data/cache/desktop-control-status.txt` → `connected=true`?
2. `GET http://127.0.0.1:8751/v1/service/status` → `state=Connected`?
3. `netstat` → **один** ESTABLISHED GUI→`:8751` (не 4+ SSE)?
4. mihomo mixed `:2181` / `:7890` LISTENING?
5. `daemon-debug.err.log` tail — `service connected` / `all probes dead`?
6. UI: API Connected, пилюля «Ошибка» → устаревший `errorText` (`internal/ui/wailsapp/display.go`, `frontend/…/Simple.tsx`).
7. «Демон не запущен» при connect → не опрашивать `/status` во время долгого `POST /connect` (см. workflowlog).

Полные команды: `AI/CURSOR_SHELL_ALLOWLIST.md` §1, §6.

---

## 6. Сборка и kit (не искать каждый раз)

| Задача | Команда |
|--------|---------|
| Daemon | `go build -o muhomor.exe ./cmd/muhomor` |
| GUI | **только** `wails build` или `scripts/build-gui-wails.ps1` — **не** `go build ./cmd/muhomor-gui` |
| Windows kit | `scripts/pack-windows-kit.ps1` → `dist/muhomor-kit/` + `.zip` |
| Linux kit | `scripts/pack-linux-kit.ps1 -TarGz` |
| Остановить процессы | `scripts/stop-muhomor.ps1` |
| Kit e2e smoke (daemon API, CI) | `scripts/kit-e2e-smoke.ps1 -MinimalPack -UILauncher daemon` |
| Kit e2e + hidden Wails | `scripts/kit-e2e-smoke.ps1 -UILauncher gui` |
| Копировать exe в kit | `go build -o dist/muhomor-kit/muhomor.exe ./cmd/muhomor` + `build-gui-wails.ps1 -OutFile dist/muhomor-kit/muhomor-gui.exe -NoPackKit` |
| Тесты | `go test ./internal/... -count=1 -timeout 120s` |

Pack всегда пересобирает `muhomor.exe` (даже `-SkipBuild`) — старый exe без `--daemon` ломает kit.

Geo в kit: `data/run/mihomo/geoip.metadb` — не удалять при pack.

---

## 7. `.cursorignore` и доступ агента

Игнорируются: `dist/`, `*.log`, `*.db`, `bin/mihomo*`.

**Без `all` / allowlist** агент **не видит** `dist/muhomor-kit/data/**` и не читает логи.

Опционально разрешить логи kit в индексе — см. `AI/CURSOR_SHELL_ALLOWLIST.md` §5.

---

## 8. Переменные окружения

| Переменная | Назначение |
|------------|------------|
| `MUHOMOR_DATA_DIR` | data вместо auto |
| `MUHOMOR_MIHOMO_BIN` | путь к mihomo (kit выставляет сам) |
| `MUHOMOR_MIHOMO_CONTROLLER_HOST` | override API host (WSL) |
| `MUHOMOR_MIXED_BIND_HOST` | bind mixed |
| `MUHOMOR_VLESS_URI` | import для debug |
| `MUHOMOR_TEST_IMPORT_URI` | Tier B в `kit-e2e-smoke.ps1` (POST `/v1/profiles/import`) |
| `MUHOMOR_GUI_START_HIDDEN` | `1`/`true` — скрытый старт Wails (или `-start-hidden`) |

---

## 9. Код по задаче (куда идти)

| Симптом / задача | Пакет / файл |
|------------------|--------------|
| Connect / BL / pretest | `internal/simplemode`, `internal/selector` |
| Daemon HTTP | `internal/controller`, `internal/api`, `internal/apiclient` |
| mihomo child / geo tail | `internal/mihomo` (`client.go`, `startup_log.go`) |
| Wails UI | `internal/ui/wailsapp`, `frontend/`, `internal/ui/presenter` |
| Data paths | `internal/paths` |
| Start daemon from GUI | `internal/platform/daemon.go`, `daemon_log.go` |
| REST контракт | `docs/PHASE4_0.md` |

Архитектура: `AGENTS.md`. Телеметрия baseline: `docs/2K_TELEMETRY.md`.

---

## 10. Чего не делать повторно

- Не grep'ить весь репо «где лог» — таблица §3.
- Не искать порт демона — **8751** (`DefaultDaemonAPIPort`).
- Не открывать `muhomor.db` из UI — только API.
- Не `go build` GUI без Wails.
- Не читать `frontend/node_modules`.
- Kit smoke: запуск из `dist/muhomor-kit/`, data = `./data`, не `%LOCALAPPDATA%`.

---

## 11. Связанные файлы

| Файл | Зачем |
|------|--------|
| `AI/workflowlog.md` | что уже чинили, Current focus |
| `AI/CURSOR_SHELL_ALLOWLIST.md` | команды + allowlist Cursor |
| `AI/CURSOR_HABITS.md` | лимиты, стартовый промпт |
| `AGENTS.md` | архитектура, build |
| `.cursor/rules/controller-api.mdc` | endpoints |
| `.cursor/rules/kit-pack.mdc` | pack kit |
