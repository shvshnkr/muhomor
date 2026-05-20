# Phase 3 — Full mode (начало)

## Сделано

| Область | Пакет |
|---------|--------|
| Hysteria2 URI → mihomo | `internal/configgen/hysteria.go` |
| Proxy chain / relay group | `internal/configgen/chain.go` |
| TUN section в YAML | `internal/configgen/tun.go` + `BuildOptions.Tun` |
| Протоколы: явный allowlist | `internal/subscription/protocol.go` |
| Route asset marker | `internal/subscription/assets.go` |
| Импорт hysteria/trojan из подписок | `subscription/updater`, `parse` |

## Использование TUN

```go
opt := configgen.DefaultBuildOptions()
opt.Tun = configgen.TunOptions{Enable: true, Stack: "system", DNSHijack: true}
```

## Chain (эксперт)

```go
configgen.BuildChainConfig(chain, opt, rules, rulesDir)
```

## Phase 3.1 (готово)

См. [PHASE3_1.md](PHASE3_1.md): inbound/DNS из store, CLI chain/settings, asset scheduler, [UNSUPPORTED_PROTOCOLS.md](UNSUPPORTED_PROTOCOLS.md).

## Дальше (Phase 4)

- Android / GUI / TUI
- Plugin sidecar для sing-box-only протоколов
