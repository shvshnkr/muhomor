#!/usr/bin/env bash
# Local parity with .github/workflows/ci.yml
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export CGO_ENABLED=0

go vet ./...
go build -o /dev/null ./cmd/muhomor

run_slice() {
  local name="$1"
  shift
  echo "go test [$name]"
  go test -count=1 -timeout 120s "$@"
}

run_slice controller-runtime ./internal/controller/... ./internal/simplemode/...
run_slice selector-standby-probe \
  ./internal/selector/... ./internal/standby/... ./internal/probe/... ./internal/profileclass/...
run_slice store-aggregate ./internal/store/... ./internal/aggregate/...
run_slice config-routing ./internal/configgen/... ./internal/routing/...
run_slice subscription-mihomo-apiclient \
  ./internal/subscription/... ./internal/mihomo/... ./internal/apiclient/...
run_slice ui-model-pseudogui-wails \
  ./internal/ui/model/... ./internal/ui/pseudogui/... ./internal/ui/wailsapp/...
run_slice paths-systemd ./internal/paths/... ./internal/systemd/...

echo "go test -tags=integration [subscription-fetch]"
go test -tags=integration -count=1 -timeout 120s ./internal/subscription/...

(
  cd frontend
  npm ci || npm install
  npm test
  npm run build
)

echo "ci-test-matrix: OK"
