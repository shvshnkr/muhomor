# Phase 3.1 — Full inbound, DNS, chain CLI, assets scheduler

## Сделано

| Область | Пакет / файл |
|---------|----------------|
| Настройки service_mode, mixed-port, inbound auth, TUN, chain IDs | `internal/store/settings.go` |
| Mixed/socks/http, allow-lan, bind-address, authentication | `internal/configgen/inbound.go` |
| DNS block (fake-ip / redir-host) | `internal/configgen/dns.go` |
| `BuildOptions` из store | `internal/configgen/settings.go` |
| Daemon flags → store | `controller.ApplyCLISettings` |
| Relay chain connect | `Runtime.StartChain`, `POST /v1/service/chain` |
| Периодическое обновление rule assets (24h) | `controller/scheduler.go` |
| Скачивание geosite yaml | `subscription/assets.go` |
| CLI: `--service-mode`, `--mixed-port`, `--proxy-auth`, `--route-quick-profile` | `cmd/muhomor/main.go` |
| CLI: `--profiles`, `--ctl chain --chain 1,2,3` | `cmd/muhomor/main.go` |
| Импорт: все схемы из allowlist + skip с причиной | `subscription/protocol.go`, `runImport` |

## CLI

```bash
# Режим proxy (mixed-port) или vpn (TUN)
./muhomor --daemon -d ~/.local/share/muhomor \
  --service-mode vpn --mixed-port 7890 --proxy-auth user:secret

# Быстрый маршрут: 0=manual 1=ru_direct 2=ru_blocked_ai
./muhomor --daemon --route-quick-profile 1

# Список профилей
./muhomor --profiles -d ~/.local/share/muhomor

# Цепочка relay (id из --profiles)
./muhomor --ctl chain --chain 3,7,12 -d ~/.local/share/muhomor
```

Inbound-учётные данные: на desktop по умолчанию без auth на `127.0.0.1` (mixed-port). На Android в Dahusim — случайный логин/пароль; в muhomor задаётся явно через settings/CLI.

## API

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/v1/service/chain` | `{"ids":[1,2,3]}` — relay chain |

## Assets

Планировщик в daemon раз в час проверяет `last_asset_update_at`; не чаще раза в 24h качает geosite yaml в `{mihomo}/ruleset/`.

## Дальше (Phase 4)

- Desktop UI (архитектура): [PHASE4_DESKTOP_UI_ARCH.md](PHASE4_DESKTOP_UI_ARCH.md)
- Android shell (позже, тот же appcore/REST)
- Plugin sidecar — [UNSUPPORTED_PROTOCOLS.md](UNSUPPORTED_PROTOCOLS.md)
