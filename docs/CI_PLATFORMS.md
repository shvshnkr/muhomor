# CI platforms (muhomor kit)

Целевые ОС portable kit vs то, что реально гоняет GitHub Actions.

## Целевые платформы (продукт)

| Платформа | Приоритет |
|-----------|-----------|
| Windows 11 x64, 10 x64 | основной GUI (Wails + WebView2) |
| Windows 10 x86, 7 x64/x86 | daemon/CLI kit, без Wails GUI |
| Windows 10 LTSC 1607 / 1809 x64 | как 10 x64 (ограничения WebView2) |
| Debian stable (bookworm + bullseye) | Linux kit |
| Ubuntu LTS 22.04 + 24.04 | Linux dev/kit |
| macOS | не поддерживаем |
| ARM64 Linux | низкий приоритет vs x86 |

## GitHub Actions (hosted)

| Job | Что покрывает |
|-----|----------------|
| `platform-linux` | Ubuntu 22.04/24.04, Debian bookworm/bullseye (container) — `go build` + smoke tests |
| `cross-build` | `windows/amd64`, `windows/386`, `linux/amd64`, `linux/386` — сборка `muhomor` без GUI |
| `platform-windows` | `windows-2022` (≈ Win11/Server2022), `windows-2019` (≈ Win10 x64) — kit e2e |
| `go-packages` | слайсы пакетов Go (не ОС) — только Ubuntu 22.04 |
| `gui-windows` | Wails amd64, только `windows-2022` |

**Нет на GitHub (только cross-build win/386 или ручной kit):** Win7, Win10 32-bit runtime, LTSC-образы 1607/1809 как отдельные VM.

Ручной прогон: Actions → **Run CI manually** или **CI** → Run workflow.
