#!/usr/bin/env bash
# CI gate: vet/build hygiene only (tests run in go-stability-matrix / integration jobs).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

patterns=(
  'C:\\Users\\'
  'debug-5ee762'
  'agentDebugLog'
  'agent_debug'
)
for pat in "${patterns[@]}"; do
  if git grep -n "$pat" -- ':!scripts/verify-publish.ps1' ':!scripts/verify-publish.sh' ':!scripts/verify-publish-ci.sh' ':!docs/PUBLISHING.md' ':!internal/controller/debug_trace_stub.go' >/dev/null 2>&1; then
    echo "Forbidden pattern in git tree: $pat" >&2
    git grep -n "$pat" -- ':!scripts/verify-publish.ps1' ':!scripts/verify-publish.sh' ':!scripts/verify-publish-ci.sh' ':!docs/PUBLISHING.md' ':!internal/controller/debug_trace_stub.go' >&2
    exit 1
  fi
done

echo "verify-publish-ci: OK"
