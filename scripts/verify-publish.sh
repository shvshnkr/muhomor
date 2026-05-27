#!/usr/bin/env bash
# Pre-push publish hygiene checks. Run from repo root.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export CGO_ENABLED=0
echo "go test ./..."
go test ./...
echo "go vet ./..."
go vet ./...

patterns=(
  'C:\\Users\\'
  'debug-5ee762'
  'agentDebugLog'
  'agent_debug'
)
for pat in "${patterns[@]}"; do
  if git grep -n "$pat" -- ':!scripts/verify-publish.ps1' ':!scripts/verify-publish.sh' ':!docs/PUBLISHING.md' >/dev/null 2>&1; then
    echo "Forbidden pattern in git tree: $pat" >&2
    git grep -n "$pat" -- ':!scripts/verify-publish.ps1' ':!scripts/verify-publish.sh' ':!docs/PUBLISHING.md' >&2
    exit 1
  fi
done

if git status --porcelain | grep -qE '^\?\? dist/|\.db'; then
  echo "WARNING: untracked dist/ or *.db present — do not add them to the commit." >&2
fi

echo "verify-publish: OK"
