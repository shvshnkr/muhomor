# ADR 001: mihomo как subprocess (Phase 0)

**Статус:** принято  
**Дата:** 2026-05-20

## Контекст

Dahusim использует sing-box через `libcore` (JNI). Целевой движок — [mihomo](https://github.com/MetaCubeX/mihomo) (Clash Meta). Нужно выбрать embed vs отдельный процесс до Phase 1.

## Решение

**Phase 1–2:** запускать `mihomo` как **отдельный subprocess** с конфигом YAML на диске и управлением через `external-controller` (HTTP на `127.0.0.1:9090`).

Контроллер `muhomor`:

1. Пишет `config.yaml` в каталог рантайма (`$XDG_RUNTIME_DIR/muhomor/` на Linux).
2. Стартует: `mihomo -f <config> -d <dir>`.
3. Reload: `PUT /configs?force=true` на API (с `Authorization: Bearer <secret>`).
4. Stop: `DELETE /connections` + graceful kill процесса.

## Обоснование

| Критерий | Subprocess | Embed |
|----------|------------|-------|
| Скорость Phase 0 PoC | ✓ один скрипт + бинарь mihomo | ✗ fork API, версии |
| Обновление ядра | ✓ пакет/бинарь отдельно | ✗ пересборка muhomor |
| Параллель с Dahusim `GuardedProcessPool` | ✓ та же модель | частично |
| systemd / отладка | ✓ `journalctl`, отдельный PID | сложнее |

## Последствия

- В PATH или `MUHOMOR_MIHOMO_BIN` должен быть бинарь `mihomo`.
- Секрет API генерируется при каждой сборке конфига и хранится только в runtime-конфиге.
- Phase 4 может пересмотреть embed для single-binary Android/desktop — отдельный ADR.

## Альтернативы (отклонены)

- **Embed libmihomo:** отложено до стабильного Phase 2 и оценки размера/лицензий.
- **sing-box subprocess:** против цели миграции на mihomo.
