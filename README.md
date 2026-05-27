# muhomor

Go-ядро миграции Dahusim → [mihomo](https://github.com/MetaCubeX/mihomo) (см. карту в DaiHusim `docs/local/MIGRATION_GO_MIHOMO_MAP.md`).

## Фазы

- **Phase 0:** ADR, R&D rule-providers, `scripts/poc-mihomo.sh`, `cmd/poc-mihomo`
- **Phase 1 (MVP):** SQLite, configgen, daemon, `--ctl`, reachability, simple connect, RU direct rules, systemd
- **Phase 2:** selector, handoff/adapt, health, bootstrap, quick routing — [docs/PHASE2.md](docs/PHASE2.md)
- **Phase 2.1:** WL builtins, pretest, exit probe, rule-providers, rtnetlink — [docs/PHASE2_1.md](docs/PHASE2_1.md)
- **Phase 3:** hysteria, chain, tun, protocol matrix — [docs/PHASE3.md](docs/PHASE3.md)
- **Phase 3.1:** inbound/DNS, service-mode, chain CLI, asset scheduler — [docs/PHASE3_1.md](docs/PHASE3_1.md)
- **Pseudo-GUI:** `--pseudo-gui` (Linux/Windows) — [docs/PSEUDOGUI.md](docs/PSEUDOGUI.md)
- **Phase 4.0:** REST API, SSE events, remote appcore — [docs/PHASE4_0.md](docs/PHASE4_0.md)
- **Phase 4.1:** Desktop GUI (Wails + React) — [docs/PHASE4_1.md](docs/PHASE4_1.md), [docs/UI_PRODUCT_BRIEF.md](docs/UI_PRODUCT_BRIEF.md)
- **Phase 4.2+ (план):** Full screens — [docs/PHASE4_DESKTOP_UI_ARCH.md](docs/PHASE4_DESKTOP_UI_ARCH.md)

## Contributing / build

- **Daemon & CLI** — no CGO: `go test ./...`, `go build ./cmd/muhomor`
- **GUI** (`muhomor-gui`) — Node 18+ + [Wails CLI](https://wails.io): `powershell -File scripts/build-gui-wails.ps1` (Windows WebView2; Linux: webkit2gtk)
- Portable kits are **not** in git; build with `scripts/pack-windows-kit.ps1` / `pack-linux-kit.ps1` → `dist/`
- Before publishing: [docs/PUBLISHING.md](docs/PUBLISHING.md), `scripts/verify-publish.ps1`
- License: [MIT](LICENSE)

## Требования

- Go 1.23+ (see `go.mod`)
- Бинарь `mihomo` в `PATH` или `MUHOMOR_MIHOMO_BIN` — Windows: `scripts/fetch-mihomo-windows.ps1` → `bin/mihomo.exe` ([релизы MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo/releases))

## Быстрый старт (Linux)

```bash
# Phase 0 PoC
export MUHOMOR_VLESS_URI='vless://...'
./scripts/poc-mihomo.sh

# Phase 1
go build -o muhomor ./cmd/muhomor
./muhomor --import-uri "$MUHOMOR_VLESS_URI"
./muhomor --daemon -d ~/.local/share/muhomor
./muhomor --ctl start -d ~/.local/share/muhomor
./muhomor --ctl status -d ~/.local/share/muhomor
```

### Portable kit (static binary)

Сборка на Windows (кросс-компиляция, `CGO_ENABLED=0`):

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\pack-linux-kit.ps1 -TarGz
```

Результат: `dist/muhomor-kit-linux-amd64.tar.gz` — `muhomor` + `bin/mihomo` + `config/` + чистая `data/` (proxy, 2181, один туннель). На Linux: распаковать, `chmod +x muhomor start.sh test-kit.sh`, `./test-kit.sh`, `./start.sh`.

## CLI (совместимость с DesktopMain)

| Флаг | Действие |
|------|----------|
| `--daemon` | фоновый процесс + Unix socket API |
| `--ctl start\|stop\|reload\|status\|ping\|chain` | управление |
| `--ctl chain --chain 1,2,3` | relay chain |
| `--profiles` | список профилей в SQLite |
| `--service-mode proxy\|vpn` | mixed-port или TUN |
| `--mixed-port N` | порт mixed inbound |
| `--proxy-auth user:pass\|none` | auth на inbound |
| `--route-quick-profile 0\|1\|2` | quick routing |
| `--pseudo-gui` | интерактивное терминальное меню |
| `--systemd install\|...` | user unit (Linux) |
| `--import-uri` / `--import-file` | импорт (vless/trojan/hysteria*) |
| `-d` / `--dir` | каталог данных |

## Документация

- [ADR 001: subprocess](docs/adr/001-mihomo-subprocess.md)
- [Rule providers R&D](docs/RULE_PROVIDERS_RND.md)
- [Protocol gap](docs/PROTOCOL_GAP.md)
