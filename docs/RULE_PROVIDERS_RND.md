# R&D: rule-providers sing-box → mihomo (Phase 0)

Источник URL в Dahusim: `SingBoxOptionsUtil.kt` (SagerNet + runetfreedom).

## URL из Kotlin

| Назначение | База URL | Файл (sing-box) |
|------------|----------|-----------------|
| Geosite общий | `https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set` | `{tag}.srs` |
| Geoip общий | `https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set` | `{tag}.srs` |
| RU blocked geosite | `.../runetfreedom/.../rule-set-geosite` | `geosite-ru-blocked*.srs` |
| RU blocked geoip | `.../runetfreedom/.../rule-set-geoip` | `geoip-ru-blocked*.srs` |

Маппинг тега `geosite-ru` → файл `geosite-category-ru.srs` (актуальный sing-geosite).

## Совместимость с mihomo

| Формат | sing-box | mihomo (Clash Meta) | Phase 1 |
|--------|----------|---------------------|---------|
| `.srs` (binary rule-set) | ✓ native | ✗ не поддерживается напрямую | не использовать |
| Clash YAML provider | опционально | ✓ `rule-providers` behavior classical/domain/ipcidr | Phase 2 blocked/AI |
| Встроенный GEOSITE/GEOIP | через rule-set tag | ✓ при наличии `geodata` (MMDB) | **Phase 1 RU direct** |

## Решение для MVP (Phase 1)

**RU direct only** (без `geosite-ru-blocked` / AI):

```yaml
rules:
  - GEOSITE,ru,DIRECT
  - GEOIP,ru,DIRECT
  - MATCH,PROXY
```

Требование: при первом запуске mihomo должен иметь актуальный GeoSite/GeoIP (стандартный механизм mihomo / `GEODATA`).

## Phase 2 (blocked + AI)

Варианты (по приоритету):

1. **Clash-совместимые** списки runetfreedom / зеркала в формате `.yaml` (если есть).
2. **Конвертер** `.srs` → classical YAML (offline job, не в hot path).
3. **rule-providers** на Loyalsoldier / metacubex geodata, если теги совпадают.

Инвариант Dahusim сохраняется: правила PROXY для blocked/AI **выше** RU DIRECT в `rules[]`.

## Проверка PoC

Скрипт `scripts/poc-mihomo.sh` и `cmd/poc-mihomo` генерируют минимальный конфиг с GEOSITE/GEOIP; на Linux с установленным mihomo — smoke `GET /version` после старта.
