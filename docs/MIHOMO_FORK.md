# Приватная копия mihomo (ядро)

## Где исходники

| Что | URL |
|-----|-----|
| Репозиторий | [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo) |
| **Ветка с Go-ядром (нужна нам)** | [**Alpha**](https://github.com/MetaCubeX/mihomo/tree/Alpha) |
| Документация | [wiki.metacubex.one](https://wiki.metacubex.one) |
| Пример конфига | [docs/config.yaml на Alpha](https://github.com/MetaCubeX/mihomo/blob/Alpha/docs/config.yaml) |

На ветке **Alpha** лежит Clash Meta / mihomo на Go (`main.go`, `tunnel/`, `adapter/`, …).  
Ветка `main` в UI GitHub может показывать другой контент — для VPN-ядра всегда берите **Alpha** или [релизы](https://github.com/MetaCubeX/mihomo/releases).

`muhomor` **не обязан** форкать mihomo для Phase 1: достаточно бинаря в `PATH` (ADR 001). Форк нужен, если планируете патчи ядра.

## Почему не «приватный форк» в классическом смысле

У публичного `MetaCubeX/mihomo` обычный **Fork** на бесплатном GitHub почти всегда тоже **публичный**.  
Чтобы хранить изменения приватно:

1. Создать **новый приватный репозиторий** (не Fork).
2. Залить туда ветку `Alpha` (+ свои ветки).
3. Добавить `upstream` на `MetaCubeX/mihomo` для `git fetch upstream`.

## Ваш репозиторий (создан)

| | |
|--|--|
| URL | **https://github.com/shvshnkr/mihomo-muhomor** (private) |
| Ветка по умолчанию | `Alpha` |
| Локальный клон | `c:\Users\user\mihomo-muhomor-src` |
| Upstream remote | `upstream` → `MetaCubeX/mihomo` |

Повторить настройку: `scripts/setup-mihomo-private.ps1`

## Шаги вручную (GitHub CLI)

После `gh auth login` — см. скрипт выше или:

```powershell
git clone --branch Alpha https://github.com/MetaCubeX/mihomo.git mihomo-src
cd mihomo-src
git remote rename origin upstream
git remote add origin https://github.com/USER/mihomo-muhomor.git
git push -u origin Alpha
```

## Связь с muhomor

```bash
export MUHOMOR_MIHOMO_BIN=/path/to/mihomo   # собранный из вашего приватного клона
```

Сборка ядра (в клоне, ветка Alpha):

```bash
go build -o mihomo
```

## Лицензия

Upstream — GPL-3.0. В README приватного репо укажите происхождение от MetaCubeX/mihomo (ветка Alpha).
