# Неподдерживаемые протоколы (muhomor → mihomo)

muhomor собирает YAML для **mihomo** (Clash Meta). Протоколы sing-box / Dahusim, которых нет в mihomo, **не импортируются** — строка пропускается с причиной в stderr.

## Поддерживаются (Phase 3.1)

| Схема | Примечание |
|-------|------------|
| `vless` | основной MVP |
| `trojan` | + WL builtin |
| `hysteria` / `hysteria2` | `configgen/hysteria.go` |

Проверка: `subscription.UnsupportedReason`, allowlist в `internal/subscription/protocol.go`.

## Не поддерживаются (явно)

Примеры схем, которые Dahusim/sing-box может иметь, но mihomo subprocess не покрывает без доработки ядра или sidecar:

| Схема | Статус |
|-------|--------|
| `ss` / `ssr` | не в allowlist |
| `vmess` | не в allowlist (зависит от сборки mihomo) |
| `wireguard` | отдельный стек |
| `tuic` | sing-box native |
| `ssh` | нет |
| `anytls`, custom plugins | только в sing-box |

При импорте подписки: `skip: … (protocol_unsupported:vmess)`.

## Plugin sidecar (план)

Для паритета с Dahusim без форка mihomo на все протоколы:

1. Отдельный процесс sing-box / кастомный outbound по `docs/PROTOCOL_GAP.md`.
2. muhomor держит chain: `local → sidecar → exit`.
3. Единый selector/store; YAML только для поддерживаемых hop'ов.

До реализации sidecar — документировать пропуски в логах импорта и не падать на всей подписке.

## См. также

- [PROTOCOL_GAP.md](PROTOCOL_GAP.md)
- [PHASE3.md](PHASE3.md)
