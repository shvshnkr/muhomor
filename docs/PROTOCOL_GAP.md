# Матрица протоколов: Dahusim → mihomo (Phase 0)

По `ProxyEntity` / `DefaultUserBootstrap.defaultSubscriptionLinks` (VLESS, Hysteria, raw lists).

| type | sing-box (Dahusim) | mihomo | MVP Phase 1 | Примечание |
|------|-------------------|--------|-------------|------------|
| socks, http | ✓ | ✓ | отложить | не в default links |
| ss, vmess, vless, trojan | ✓ | ✓ | **vless** | PoC + configgen |
| hysteria, tuic, wireguard | ✓ | ✓ | hysteria Phase 1.1 | парсер подписки позже |
| ssh, direct | ✓ | частично | нет | |
| naive, mieru, juicity, shadowquic | plugin sidecar | ✗ | **нет** | explicit unsupported |
| anytls, trusttunnel | sing-box fmt | ✗ | нет | |
| chain, proxy-set | ConfigBuilder | proxy-groups | Phase 2 | |
| CONFIG (raw JSON) | passthrough | import | Phase 2 | |

## MVP-подписки (из Kotlin)

Ссылки без секретов — только для теста парсера; живые узлы не гарантированы:

- `https://mifa.world/vless`
- `https://mifa.world/hysteria`
- raw `.txt` / gist списки (VLESS построчно)

**Phase 1 scope:** парсинг `vless://` из URI и raw-текста; один лучший профиль; hysteria — заглушка с ошибкой «unsupported in MVP».

## Риски

- Плагин-протоколы в подписках → пропуск с логом `protocol_unsupported`.
- Hysteria2 в mihomo — отдельный тип `hysteria2`; добавить в Phase 1.1 после VLESS stable.
