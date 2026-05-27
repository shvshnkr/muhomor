MUHOMOR — portable kit (Linux amd64/arm64)
==========================================

Состав:
  muhomor          — статический бинарь (CGO_ENABLED=0), без установки в систему
  muhomor-gui      — desktop GUI (Wails), если собран в kit (нужен webkit2gtk)
  bin/mihomo       — движок прокси
  config/          — subscriptions.txt (подписки для первого запуска)
  data/            — настройки по умолчанию (proxy, порт 2181, один туннель)

По умолчанию: режим Proxy, mixed-port 2181, один туннель, без пула load-balance.

GUI (если есть muhomor-gui):
  sudo apt install libwebkit2gtk-4.1-0   # Debian/Ubuntu runtime
  # для сборки GUI на хосте: libwebkit2gtk-4.1-dev, pkg-config, gcc
  ./muhomor-gui -d "$(pwd)/data" --service-mode proxy --mixed-port 2181

Запуск (CLI):
  chmod +x muhomor start.sh test-kit.sh bin/mihomo
  [ -f muhomor-gui ] && chmod +x muhomor-gui
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

Пул туннелей и multipath — в muhomor-gui (вкладка Настройки), pseudo-GUI [s], или API.
