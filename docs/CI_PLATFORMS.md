# CI platforms (GitHub Actions)

Что гоняет workflow **CI** на push/PR (код) и вручную (**Run CI manually** / **CI** → Run workflow).

| Job | Платформы / артефакты |
|-----|------------------------|
| `gate` | Ubuntu 22.04 — vet, frontend build, `go build` daemon |
| `go-packages` | Ubuntu 22.04 — полные `go test` по слайсам пакетов |
| `go-regression` | Ubuntu 22.04 — `go test -run Regression ./internal/...` |
| `go-integration-matrix` | Ubuntu 22.04 — `-tags integration` |
| `platform-linux-ubuntu` | Ubuntu **22.04**, **24.04** — build + smoke tests |
| `platform-linux-debian` | Debian **bookworm**, **bullseye** (container) — build + smoke tests |
| `cross-build` | сборка `muhomor`: `windows/amd64`, `windows/386`, `linux/amd64`, `linux/386` |
| `ui-frontend` | Ubuntu 22.04 — vitest + vite build |
| `platform-windows` | **windows-2022**, **windows-2025** (Server x64) — kit e2e |
| `gui-windows` | **windows-2022** — Wails build (опционально при manual + флаг) |

Лейблы GitHub: `windows-latest` = Server 2025. `windows-2019` снят (2025).

Слайс `go-packages` — это **пакеты Go**, не ОС (controller-runtime, ui-model-pseudogui-wails, …).
