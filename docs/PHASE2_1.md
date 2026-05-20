# Phase 2.1 — завершение parity

## Сделано

| Задача | Реализация |
|--------|------------|
| WL builtin pool | `internal/bootstrap/wlbuiltin.go` — 4 Trojan + WL vless (как Dahusim) |
| Trojan configgen | `internal/configgen/trojan.go` |
| Pre-connect url-test | `internal/selector/pretest.go` — ephemeral mihomo subprocess |
| WhitelistRuRouting | `internal/routing/whitelist_ru.go` — RU DIRECT→PROXY при exit RU |
| Exit probe | `internal/routing/exit_probe.go` — HTTP через mixed-port |
| Rule-providers | `internal/configgen/ruleproviders.go` — meta-rules-dat YAML |
| rtnetlink handoff | `internal/simplemode/netmon_rtnetlink.go` (Linux) |

## Проверка

```bash
export MUHOMOR_MIHOMO_BIN=/path/to/mihomo
muhomor --daemon -d ~/.local/share/muhomor
muhomor --ctl start
# WL-only сеть: в логах pool с Built-in helpers
muhomor --ctl export-log
```

## Заметки

- Trojan password встроен как в APK Dahusim — обновлять при отзыве узлов.
- Rule-providers тянутся с [meta-rules-dat](https://github.com/MetaCubeX/meta-rules-dat); при 404 см. `docs/RULE_PROVIDERS_RND.md`.
