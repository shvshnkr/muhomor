# Регрессионные тесты по «болям»

Имена `TestRegression_*` — якоря на прошлые инциденты (см. `AI/workflowlog.md`). Детальная логика остаётся в пакетах рядом с кодом; этот файл — реестр «боль → тест».

| Боль | Симптом | Тест(ы) |
|------|---------|---------|
| Ложный «демон не запущен» | timeout/SSE blip во время connect | `apiclient` `TestRegression_*`, `presenter` `TestRegression_DaemonBlip_*` |
| Гонка connect при Stop | connect во время `Stopping` | `controller` `TestRegression_QueuedConnectAfterStop`, `TestRegression_StartWhileStopping_*` |
| Picker :8760 | дубли subprocess, Ensure под нагрузкой | `mihomo` `TestRegression_PickerEnsure_*`, `selector` `TestColdPrepareProbes_*` |
| H3 delay 503 | ложный fail connect | `controller` `TestPostConnectDelay_retryOn503ThenSuccess`, `model` `TestFriendlyDelayError_503` |
| Traffic reload | x125, WARN на idle, сброс при reload | `controller` `TestRegression_SkipTrafficSample_*`, `traffic_cache_test` |
| Подписка 402/403 | HTTP от провайдера | `subscription` `TestRegression_SubscriptionHTTP_*` |
| Verifier / degraded | ложный connected | `controller` `TestConnectionVerifier*`, `selector` `TestCompositeScore_*` |
| Shared HTTP transport | утечка TCP на :8751 | `apiclient` `TestRegression_InitHTTP_*`, `TestRegression_StreamEvents_*` |
| Probe/activity UI | дубль pill vs title | `presenter` `TestRegression_FormatProbeProgress_*`, `model` `activity_ui_test` |
| Picker bind :8760 | reuse API вместо 2-го процесса | `mihomo` `TestRegression_IsPickerBindError`, `TestRegression_PickerAttachExisting_*` |
| Restart storm | H3-restart-window | `controller` `TestRegression_RecordRestartAttempt_*`, `TestRegression_BumpDegradedCycle_*` |
| Wails quit flash | ошибки при выходе | `wailsapp` `TestRegression_SanitizeConnectionForQuit_*` |
| Wails start hidden | tray smoke без окна | `wailsapp` `TestRegression_StartHiddenEnabled_*`, `-start-hidden` |
| Pseudo toast parity | error pin vs Wails | `pseudogui` `TestRegression_SyncSimpleToast_*`, `TestRegression_PickList_*` |
| Kit e2e (daemon API) | connect/stop без GUI | `scripts/kit-e2e-smoke.ps1`, CI `kit-e2e-windows` |
| Simple connect budget | deferred subs refresh | `simplemode` `TestRegression_ConnectRefreshBudget_*` |

Локально:

```powershell
go test -count=1 -run Regression ./internal/...
go test -count=1 ./internal/...
scripts/ci-test-matrix.ps1
```

GitHub Actions (workflow `CI`, push/PR **только при изменении кода** — `paths-ignore` для `docs/`, `AI/`, `*.md`; `workflow_dispatch` — полный прогон):

- `go-regression` — `go test -run Regression ./internal/...`
- `go-packages` — слайсы пакетов (Ubuntu 22.04)
- `platform-linux-*` — Ubuntu 22.04/24.04, Debian bookworm/bullseye
- `cross-build` — win/linux `386`+`amd64`
- `platform-windows` — Win10/11 x64 (`windows-2019`/`2022`), kit e2e
- См. `docs/CI_PLATFORMS.md`
- `ui-frontend`, `go-integration-matrix`, `gate`

## Три контура (kit-dual-ui-testing)

| Контур | Что проверяет | Команда |
|--------|---------------|---------|
| **1. Daemon API** | connect/stop, ctl, логи (основной e2e/CI) | `scripts/kit-e2e-smoke.ps1 -UILauncher daemon` |
| **2. Wails** | unit `TestRegression_*` в `wailsapp`; опционально hidden GUI | `-UILauncher gui`, `-start-hidden` |
| **3. Pseudo-GUI** | unit `pseudogui` `TestRegression_*`; e2e = тот же API (без TTY) | `-UILauncher pseudo` (= daemon API) |

```powershell
# CI parity (minimal pack, Tier A)
powershell -File scripts/kit-e2e-smoke.ps1 -MinimalPack -UILauncher daemon

# Полный connect (Tier B)
$env:MUHOMOR_TEST_IMPORT_URI = 'vless://...'
powershell -File scripts/kit-e2e-smoke.ps1 -UILauncher daemon

# Скрытый Wails (процесс + тот же REST)
powershell -File scripts/kit-e2e-smoke.ps1 -UILauncher gui
```

Kit file smoke: `dist\muhomor-kit\Test-Kit.bat`. Скрытый GUI: `Start-hidden.bat`. Демон вручную: `muhomor.exe --daemon -dir .\data` из корня kit.
