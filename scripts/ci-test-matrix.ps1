# Local parity with .github/workflows/ci.yml (Go slices + integration + frontend).
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Push-Location $Root
$env:CGO_ENABLED = "0"

$slices = @(
    @{
        Name     = "controller-runtime"
        Packages = @("./internal/controller/...", "./internal/simplemode/...")
    },
    @{
        Name     = "selector-standby-probe"
        Packages = @(
            "./internal/selector/...",
            "./internal/standby/...",
            "./internal/probe/...",
            "./internal/profileclass/..."
        )
    },
    @{
        Name     = "store-aggregate"
        Packages = @("./internal/store/...", "./internal/aggregate/...")
    },
    @{
        Name     = "config-routing"
        Packages = @("./internal/configgen/...", "./internal/routing/...")
    },
    @{
        Name     = "subscription-mihomo-apiclient"
        Packages = @(
            "./internal/subscription/...",
            "./internal/mihomo/...",
            "./internal/apiclient/..."
        )
    },
    @{
        Name     = "ui-model-pseudogui-wails"
        Packages = @(
            "./internal/ui/model/...",
            "./internal/ui/presenter/...",
            "./internal/ui/pseudogui/...",
            "./internal/ui/wailsapp/..."
        )
    },
    @{
        Name     = "paths-systemd"
        Packages = @("./internal/paths/...", "./internal/systemd/...")
    }
)

Write-Host "go vet ./..."
go vet ./...
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }

Write-Host "go build ./cmd/muhomor"
go build -o $null ./cmd/muhomor
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }

foreach ($slice in $slices) {
    Write-Host "go test [$($slice.Name)]"
    go test -count=1 -timeout 120s @($slice.Packages)
    if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
}

Write-Host "go test -run Regression [pain anchors]"
go test -count=1 -timeout 120s -run Regression ./internal/...
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }

Write-Host "go test -tags=integration [subscription-fetch]"
go test -tags=integration -count=1 -timeout 120s ./internal/subscription/...
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }

Push-Location frontend
Write-Host "npm test (frontend)"
npm test
if ($LASTEXITCODE -ne 0) { Pop-Location; Pop-Location; exit $LASTEXITCODE }
Write-Host "npm run build (frontend)"
npm run build
if ($LASTEXITCODE -ne 0) { Pop-Location; Pop-Location; exit $LASTEXITCODE }
Pop-Location

Write-Host "ci-test-matrix: OK"
Pop-Location
