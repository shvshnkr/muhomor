# White Bolt WL (БС-сеть)

Каталог [russian-white-bolt](https://gitverse.ru/RUVIPIEN/russian-white-bolt) — агрегатор **White** подписок для whitelist-only сети (БС). **Black** подписки для обычного интернета остаются в `DefaultLinks` (`internal/subscription/bootstrap.go`).

## Два пула

| Сеть | Подписки | Bootstrap |
|------|----------|-----------|
| БС (`WhitelistOnly`) | `White Bolt WL: *` + builtin WL | `BootstrapWhiteBoltWL` |
| Обычная (нет БС) | `Quick Subscription N` (Black/open) | `Bootstrap` |

## Источники White Bolt

См. `WhiteBoltWLSources` в `internal/subscription/whitebolt_manifest.go`. Black-URL из white-bolt (igareck BLACK, SilentGhost BlackList, wlrus blackl) **не** дублируются.

## Поведение при connect

1. После probe: на БС вызывается `BootstrapWhiteBoltWL` (идемпотентно).
2. Refresh: `RefreshDueWL` на БС, `RefreshDueOpen` на обычной сети (параллельно, с таймаутом 8 с / 2.8 с).
3. Профили White Bolt получают `whitelist_marked=1` (приоритет в `selector.buildPool`).
4. Импорт обрезается до 400 URI на группу.

Планировщик по-прежнему может вызывать полный `RefreshDue` для всех групп.
