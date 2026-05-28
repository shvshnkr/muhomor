# muhomor — guide for AI agents

Go migration Dahusim → [mihomo](https://github.com/MetaCubeX/mihomo). Module: `github.com/muhomor/muhomor`.

## GitHub repositories (не менять имена без явного запроса)

| Repo | Роль |
|------|------|
| [shvshnkr/muhomor](https://github.com/shvshnkr/muhomor) | **Приложение** (этот репо): daemon, Wails, CI |
| [shvshnkr/muhomor-mihomo](https://github.com/shvshnkr/muhomor-mihomo) | **Приватное ядро** mihomo (ветка Alpha) |

**Запрет:** не делать swap (`muhomor` ↔ ядро, app → `muhomor-app`). Решение зафиксировано 2026-05-28; см. отменённый план `.cursor/plans/github_repo_swap_b87c03a4.plan.md`.

## Architecture (3 processes)

```
muhomor-gui (Fyne, CGO)  ──HTTP/SSE──►  muhomor --daemon  ──subprocess──►  mihomo
muhomor CLI (--pseudo-gui, --ctl)  ──────────┘
```

- **Daemon** owns SQLite, TUN/proxy, mihomo PID — single writer.
- **GUI/CLI** talk to daemon via `internal/apiclient`; domain logic in `internal/appcore`.
- **mihomo** is always a subprocess with on-disk YAML (ADR: `docs/adr/001-mihomo-subprocess.md`).

## Where to change what

| Task | Packages / paths |
|------|------------------|
| Wails GUI | `cmd/muhomor-gui`, `internal/ui/wailsapp`, `frontend/`, `wails.json` |
| Presenter / UI model | `internal/ui/presenter`, `internal/ui/model` (shared Fyne/Wails) |
| Fyne UI (deprecated) | `internal/ui/fyneapp` — `-tags fyne` only |
| Terminal UI | `internal/ui/pseudogui` |
| Daemon / REST API | `internal/controller`, `internal/apiclient`, `internal/api` |
| Connect / adapt / health | `internal/simplemode`, `internal/selector` |
| Config / YAML | `internal/configgen`, `internal/mihomo` |
| Profiles / subs | `internal/store`, `internal/subscription`, `internal/profiles` |
| Portable kit | `kit/`, `scripts/pack-*.ps1`, `internal/paths` |
| Binaries | `cmd/muhomor`, `cmd/muhomor-gui` |

## Build

```powershell
go build -o muhomor.exe ./cmd/muhomor
# GUI: Node 18+ and Wails CLI (go install github.com/wailsapp/wails/v2/cmd/wails@latest)
wails build -platform windows/amd64 -o muhomor-gui.exe
# Or: powershell -File scripts/build-gui-wails.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\pack-windows-kit.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\pack-linux-kit.ps1 -TarGz
```

GUI: **Wails** (Windows WebView2; Linux: webkit2gtk + CGO). Legacy Fyne: `-tags fyne` + gcc. Daemon: **no CGO**.

## Invariants

- UI never opens SQLite directly — only via daemon API / appcore.
- Do not embed mihomo in Go; lifecycle in `internal/mihomo` only.
- Portable kit: `.muhomor-portable` + `bin/mihomo` → data under `./data`, mixed port **2181**.
- Wails UI: bindings in `wailsapp`, no direct SQLite; events via `runtime.EventsEmit`.
- Legacy Fyne: `//go:build fyne` + `*_windows.go` / `*_stub.go` in `fyneapp`.

## Session handoff

Before large changes read **`AI/workflowlog.md`** (section **Current focus** + last date entry) and **`AI/AGENT_MAP.md`** (data dirs, log paths, kit vs desktop, ports, «не искать снова»).

After significant edits append a short bullet to `AI/workflowlog.md` (see `.cursor/rules/ai-workflow-log.mdc`).

## Skills (invoke explicitly)

- `/pack-kit` — assemble Windows/Linux portable kit
- `/fyne-window-fix` — Windows Fyne window layout / chrome checklist

## Do not index or edit

`dist/`, `muhomor-kit-*`, `tmp-*`, `*.db`, `bin/mihomo*`, runtime logs — see `.cursorignore`.

## Docs (pointers only)

- Overview: `README.md`
- Desktop UI plan: `docs/PHASE4_DESKTOP_UI_ARCH.md`
- REST API: `docs/PHASE4_0.md`
- Cursor habits: `AI/CURSOR_HABITS.md`
- Agent log/data map: `AI/AGENT_MAP.md`
