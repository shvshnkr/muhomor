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
- **Phase 4 (план):** desktop UI — [docs/PHASE4_DESKTOP_UI_ARCH.md](docs/PHASE4_DESKTOP_UI_ARCH.md)

## Требования

- Go 1.22+
- Бинарь `mihomo` в `PATH` или `MUHOMOR_MIHOMO_BIN` (собрать из [приватного клона Alpha](docs/MIHOMO_FORK.md) или [MetaCubeX/mihomo Alpha](https://github.com/MetaCubeX/mihomo/tree/Alpha))

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
