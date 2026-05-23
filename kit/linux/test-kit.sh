#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
FAIL=0
check() { if "$@"; then echo "[OK] $*"; else echo "[FAIL] $*"; FAIL=1; fi }

test -x "$ROOT/muhomor" || { echo "[FAIL] muhomor not executable (chmod +x muhomor)"; FAIL=1; }
test -f "$ROOT/bin/mihomo" || { echo "[FAIL] bin/mihomo missing"; FAIL=1; }
test -f "$ROOT/.muhomor-portable" || { echo "[FAIL] .muhomor-portable missing"; FAIL=1; }
test -f "$ROOT/data/muhomor.db" || { echo "[FAIL] data/muhomor.db missing"; FAIL=1; }

# Повторный tar -xzf перезаписывает muhomor.db, но оставляет старые -wal/-shm → SQLite malformed (11).
if test -f "$ROOT/data/muhomor.db-wal" || test -f "$ROOT/data/muhomor.db-shm"; then
  rm -f "$ROOT/data/muhomor.db-wal" "$ROOT/data/muhomor.db-shm"
  echo "[INFO] removed stale muhomor.db-wal/shm"
fi

if test -f "$ROOT/bin/mihomo"; then
  chmod +x "$ROOT/bin/mihomo" 2>/dev/null || true
  "$ROOT/bin/mihomo" -v || "$ROOT/bin/mihomo" 2>&1 | head -1
fi

if test -x "$ROOT/muhomor"; then
  "$ROOT/muhomor" --ctl ping -d "$ROOT/data" 2>/dev/null && echo "[OK] daemon ping" || echo "[INFO] daemon not running (start with ./start.sh)"
fi

exit "$FAIL"
