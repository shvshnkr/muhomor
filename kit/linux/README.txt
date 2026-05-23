MUHOMOR — portable kit (Linux amd64/arm64)
==========================================

Состав:
  muhomor          — статический бинарь (CGO_ENABLED=0), без установки в систему
  bin/mihomo       — движок прокси
  config/          — subscriptions.txt (подписки для первого запуска)
  data/            — настройки по умолчанию (proxy, порт 2181, один туннель)

По умолчанию: режим Proxy, mixed-port 2181, один туннель, без пула load-balance.

Запуск:
  chmod +x muhomor start.sh test-kit.sh bin/mihomo
  ./test-kit.sh
  ./start.sh              # pseudo-GUI в терминале

Если после повторного распаковывания архива ошибка «database disk image is malformed (11)»:
  rm -f data/muhomor.db-wal data/muhomor.db-shm
  (или восстановите data/muhomor.db из tar — ./test-kit.sh удалит stale WAL сам)
  ./muhomor --daemon -d "$(pwd)/data" --service-mode proxy --mixed-port 2181
  ./muhomor --pseudo-gui -d "$(pwd)/data" --mixed-port 2181

Переменные (опционально):
  export MUHOMOR_DATA_DIR="$(pwd)/data"
  export MUHOMOR_MIHOMO_BIN="$(pwd)/bin/mihomo"

Пул туннелей и multipath — в меню [s] Настройки (pseudo-GUI) или через API.
