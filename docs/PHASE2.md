# Phase 2 — Dahusim parity (desktop)

## Реализовано

| Область | Пакет | Статус |
|---------|-------|--------|
| AutoServerSelector (TCP, rank, fallback, cooldown, WL pool) | `internal/selector` | ✓ subset |
| Simple connect + subscription refresh budget | `internal/simplemode/connect` | ✓ |
| Handoff adapt (debounce, supersede) | `internal/simplemode/adapt` | ✓ |
| Session health + fallback | `internal/simplemode/health` | ✓ |
| Post-connect maintenance guard | `internal/simplemode/maintenance` | ✓ |
| Reachability grace cache | `internal/simplemode/reachability_cache` | ✓ |
| Netmon Linux (poll) | `internal/simplemode/netmon_linux` | ✓ MVP |
| Route quick profile rules order | `internal/routing` | ✓ |
| DefaultUserBootstrap links | `internal/subscription/bootstrap` | ✓ |
| Offline subscription guard | `internal/subscription/updater` | ✓ |
| ctl export-log, update-check stub | `internal/controller` | ✓ |
| API `/v1/simple/adapt` | daemon | ✓ |

## Ещё не портировано (Phase 2.1 / 3)

- Полный url-test через временный libcore-конфиг до первого connect
- Builtin Trojan WL pool (`WhitelistBuiltinBootstrap`)
- `WhitelistRuRouting` + exit probe
- Rule-providers `.srs` → clash YAML для blocked/AI
- rtnetlink netmon (сейчас poll 2s)
- App update install

## Критерий проверки

```bash
muhomor --daemon -d ~/.local/share/muhomor
muhomor --ctl start
# смена Wi‑Fi → лог H30 network handoff → adapt
muhomor --ctl status
```
