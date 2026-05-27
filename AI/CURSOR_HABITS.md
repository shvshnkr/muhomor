# Cursor: экономия лимитов (ежедневные привычки)

Настройки Cursor (не в git) — выполнить вручную один раз:

1. **Default model** → **Auto** или **Composer 2.5** (Settings → Models).
2. **Max Mode** → выключить для рутины.
3. **User Rules** → заменить на сокращённый текст из `AI/user-rules-trimmed.md`.

## На каждую задачу

| Шаг | Режим | Зачем |
|-----|-------|-------|
| Понять код | **Ask** (Shift+Tab) | Без tool loops |
| Спроектировать multi-file | **Plan** | Один чат на дизайн |
| Править 1–3 файла | **Agent** + Auto | Дешёвый пул |
| Готово | **Новый чат** | Не тащить историю |

## Стартовый промпт (копировать в новый чат)

```text
Прочитай AGENTS.md, AI/AGENT_MAP.md и AI/workflowlog.md (Current focus).
Задача: [одна вещь].
Файлы: @[path]
Не трогать: [область]
После правки: [build/check] и запись в workflowlog.
```

Шаблоны: `AI/prompts/`.

## Не делать

- Один Agent-чат на UI + kit + controller одновременно.
- `@Past Chats`, `@Branch`, `@Terminals` без нужды.
- «Найди сам по всему репо» — указывайте `@file` или пакет.
