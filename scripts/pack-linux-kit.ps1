# Portable Linux kit: static muhomor + mihomo + config + seeded data/.
# Run from repo root:  powershell -ExecutionPolicy Bypass -File .\scripts\pack-linux-kit.ps1
param(
    [string]$OutDir = "",
    [ValidateSet("amd64", "arm64")]
    [string]$Arch = "amd64",
    [switch]$TarGz,
    [switch]$SkipBuild,
    [string]$MihomoVersion = "v1.19.25"
)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
if ($OutDir -eq "") {
    $OutDir = Join-Path $Root "dist\muhomor-kit-linux-$Arch"
}

$goos = "linux"
$goarch = $Arch

if (-not $SkipBuild) {
    Write-Host "Building static muhomor ($goos/$goarch)..."
    Push-Location $Root
    $env:CGO_ENABLED = "0"
    $env:GOOS = $goos
    $env:GOARCH = $goarch
    $outBin = Join-Path $OutDir "muhomor"
    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
    go build -trimpath -ldflags "-s -w" -o $outBin ./cmd/muhomor
    Pop-Location
    Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
}

$Mihomo = Join-Path $Root "bin\mihomo"
if (-not (Test-Path $Mihomo)) {
    Write-Host "Fetching mihomo ($Arch)..."
    & (Join-Path $Root "scripts\fetch-mihomo-linux.ps1") -Version $MihomoVersion -Arch $Arch
}

New-Item -ItemType Directory -Force -Path $OutDir, (Join-Path $OutDir "bin") | Out-Null
Copy-Item (Join-Path $Root "bin\mihomo") (Join-Path $OutDir "bin\mihomo") -Force
if (-not (Test-Path (Join-Path $OutDir "muhomor"))) {
    throw "muhomor binary missing in $OutDir (build failed?)"
}

$kitLinux = Join-Path $Root "kit\linux"
foreach ($f in @("start.sh", "test-kit.sh", "README.txt")) {
    $src = Join-Path $kitLinux $f
    if (Test-Path $src) { Copy-Item $src (Join-Path $OutDir $f) -Force }
}

$kitConfigSrc = Join-Path $Root "kit\config"
$configDst = Join-Path $OutDir "config"
if (Test-Path $kitConfigSrc) {
    if (Test-Path $configDst) { Remove-Item -Recurse -Force $configDst }
    Copy-Item -Recurse -Force $kitConfigSrc $configDst
}

$ver = try { git -C $Root rev-parse --short HEAD 2>$null } catch { "unknown" }
Set-Content -Path (Join-Path $OutDir "VERSION.txt") -Value "muhomor-kit-linux-$Arch`ncommit=$ver`nbuilt=$(Get-Date -Format o)" -Encoding UTF8
New-Item -ItemType File -Force -Path (Join-Path $OutDir ".muhomor-portable") | Out-Null

$dataDir = Join-Path $OutDir "data"
if (Test-Path $dataDir) { Remove-Item -Recurse -Force $dataDir }
Write-Host "Initializing kit data..."
Push-Location $Root
go run ./scripts/init-kit-data -o $dataDir
if ($LASTEXITCODE -ne 0) { Pop-Location; throw "init-kit-data failed" }
go run ./scripts/verify-kit-settings -db (Join-Path $dataDir "muhomor.db")
if ($LASTEXITCODE -ne 0) { Pop-Location; throw "verify-kit-settings failed" }
Pop-Location
foreach ($junk in @(
    "gui.lock", "muhomor.db-wal", "muhomor.db-shm",
    "cache\logs-history", "cache\pretest-last-path.txt", "cache\pretest-last.yaml",
    "run\config.yaml", "run\mihomo\geoip.metadb"
)) {
    $p = Join-Path $dataDir $junk
    if (Test-Path -LiteralPath $p) { Remove-Item -LiteralPath $p -Recurse -Force -ErrorAction SilentlyContinue }
}

Write-Host ""
Write-Host "Kit ready: $OutDir"
Get-ChildItem $OutDir | Format-Table Name, Length -AutoSize

if ($TarGz) {
    $archive = "$OutDir.tar.gz"
    if (Test-Path $archive) { Remove-Item $archive -Force }
    tar -czf $archive -C (Split-Path $OutDir -Parent) (Split-Path $OutDir -Leaf)
    Write-Host "Archive: $archive"
}
