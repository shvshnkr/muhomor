#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
export MUHOMOR_MIHOMO_BIN="${MUHOMOR_MIHOMO_BIN:-$ROOT/bin/mihomo}"
export MUHOMOR_DATA_DIR="${MUHOMOR_DATA_DIR:-$ROOT/data}"
exec "$ROOT/muhomor" --pseudo-gui -d "$MUHOMOR_DATA_DIR" --service-mode proxy --mixed-port 2181 "$@"
