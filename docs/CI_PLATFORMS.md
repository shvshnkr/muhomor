# CI platforms (GitHub Actions)

Репозиторий **private** — минуты Actions ограничены (Free ~2000 мин/мес; Windows ×2).

## push / pull_request (по умолчанию)

Один job **`ci-lite`** на `ubuntu-22.04` (~15–25 мин **суммарно**, не 15 параллельных runners):

- frontend: vitest + build + embed
- `go vet`, `go build` daemon
- `go test ./internal/...` + `TestRegression_*`
- cross-build: win/linux amd64+386 (в том же job)
- `verify-publish-ci.sh`

Без: матрицы Ubuntu/Debian, Windows e2e, Wails, integration (сеть).

## workflow_dispatch → profile **full**

Расширенная матрица (много минут): gate, 7× go-packages, linux Ubuntu/Debian, cross-build×4, Windows 2022/2025 kit e2e, опционально Wails.

Запуск: Actions → **CI** → Run workflow → `profile: lite` (дешево) или `full`.

## Без оплаты / лимита

| Вариант | Эффект |
|---------|--------|
| Только **ci-lite** на push | Минимум минут |
| **profile=full** только вручную | Редко |
| Репо **public** | Hosted minutes для public — бесплатно без лимита минут |
| Billing: лимит $0 | Даже free minutes могут блокироваться — Settings → Billing |
| Локально | `go test ./internal/...`, `scripts/kit-e2e-smoke.ps1` на своём ПК |
