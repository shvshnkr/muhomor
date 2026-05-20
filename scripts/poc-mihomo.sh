#!/usr/bin/env bash
# Phase 0 PoC: VLESS → YAML → mihomo (Linux).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
URI="${MUHOMOR_VLESS_URI:-${1:-}}"

if [[ -z "$URI" ]]; then
  echo "Usage: MUHOMOR_VLESS_URI='vless://...' $0" >&2
  echo "   or: $0 'vless://...'" >&2
  exit 2
fi

if ! command -v mihomo >/dev/null 2>&1; then
  echo "mihomo not found in PATH. Install MetaCubeX/mihomo release binary." >&2
  exit 1
fi

cd "$ROOT"
go run ./cmd/poc-mihomo -uri "$URI" -run -wait 10s
