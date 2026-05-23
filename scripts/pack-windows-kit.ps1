# Assemble portable Windows kit: exe + mihomo + launchers + Russian help.
# Run from repo root:  powershell -ExecutionPolicy Bypass -File .\scripts\pack-windows-kit.ps1
param(
    [string]$OutDir = "",
    [switch]$Zip,
    [switch]$SkipBuild
)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
if ($OutDir -eq "") {
    $OutDir = Join-Path $Root "dist\muhomor-kit"
}

$GccUcrt = "C:\msys64\ucrt64\bin"
$GccMingw = "C:\msys64\mingw64\bin"
if (Test-Path "$GccUcrt\gcc.exe") { $env:Path = "$GccUcrt;$env:Path" }
elseif (Test-Path "$GccMingw\gcc.exe") { $env:Path = "$GccMingw;$env:Path" }

if (-not $SkipBuild) {
    Write-Host "Building muhomor.exe..."
    Push-Location $Root
    go build -o muhomor.exe ./cmd/muhomor
    $env:CGO_ENABLED = "1"
    Write-Host "Building muhomor-gui.exe (CGO)..."
    # -H windowsgui: no extra console window; closing it must not kill the GUI
    go build -tags cgo -ldflags "-H windowsgui" -o muhomor-gui.exe ./cmd/muhomor-gui
    Pop-Location
}

$Mihomo = Join-Path $Root "bin\mihomo.exe"
if (-not (Test-Path $Mihomo)) {
    Write-Host "Fetching mihomo..."
    & (Join-Path $Root "scripts\fetch-mihomo-windows.ps1")
}

New-Item -ItemType Directory -Force -Path $OutDir, (Join-Path $OutDir "bin") | Out-Null

Copy-Item (Join-Path $Root "muhomor.exe") $OutDir -Force
Copy-Item (Join-Path $Root "muhomor-gui.exe") $OutDir -Force
Copy-Item $Mihomo (Join-Path $OutDir "bin\mihomo.exe") -Force

$kitFiles = @(
    "СПРАВКА.txt",
    "Start-WG-mode3.bat", "Start-normal.bat", "Test-Kit.bat",
    "Запуск-WG-режим3.bat", "Запуск-обычный.bat",
    "Rotate-Logs.bat", "Ротация-логов.bat"
)
foreach ($f in $kitFiles) {
    $src = Join-Path $Root "dist\muhomor-kit\$f"
    $dst = Join-Path $OutDir $f
    if (-not (Test-Path -LiteralPath $src)) { continue }
    if ((Resolve-Path -LiteralPath $src).Path -ne (Resolve-Path -LiteralPath $dst -ErrorAction SilentlyContinue).Path) {
        Copy-Item -LiteralPath $src $OutDir -Force
    }
}
# UTF-8 BOM for Russian help (Notepad); bat console text stays ASCII
$utf8 = New-Object System.Text.UTF8Encoding $true
$helpRu = Join-Path $OutDir "СПРАВКА.txt"
if (Test-Path -LiteralPath $helpRu) {
    $c = Get-Content -LiteralPath $helpRu -Raw -Encoding UTF8
    [System.IO.File]::WriteAllText($helpRu, $c, $utf8)
    [System.IO.File]::WriteAllText((Join-Path $OutDir "SPRAVKA.txt"), $c, $utf8)
}

$ver = try { git -C $Root rev-parse --short HEAD 2>$null } catch { "unknown" }
Set-Content -Path (Join-Path $OutDir "VERSION.txt") -Value "muhomor-kit`ncommit=$ver`nbuilt=$(Get-Date -Format o)" -Encoding UTF8
# Portable marker: exe auto-use .\data and bin\mihomo (no %LOCALAPPDATA% leak)
New-Item -ItemType File -Force -Path (Join-Path $OutDir ".muhomor-portable") | Out-Null

# Fresh data/ + config/ for new users (no developer logs or profiles)
$kitConfigSrc = Join-Path $Root "kit\config"
$configDst = Join-Path $OutDir "config"
if (Test-Path $kitConfigSrc) {
    if (Test-Path $configDst) { Remove-Item -Recurse -Force $configDst }
    Copy-Item -Recurse -Force $kitConfigSrc $configDst
}
$dataDir = Join-Path $OutDir "data"
if (Test-Path $dataDir) { Remove-Item -Recurse -Force $dataDir }
Write-Host "Initializing kit data (default settings DB)..."
Push-Location $Root
go run ./scripts/init-kit-data -o $dataDir
if ($LASTEXITCODE -ne 0) { Pop-Location; throw "init-kit-data failed" }
go run ./scripts/verify-kit-settings -db (Join-Path $dataDir "muhomor.db")
if ($LASTEXITCODE -ne 0) { Pop-Location; throw "verify-kit-settings failed" }
Pop-Location
# Drop dev leftovers if data/ could not be fully removed (locked files)
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
Get-ChildItem $OutDir, (Join-Path $OutDir "bin") | Format-Table Name, Length -AutoSize

if ($Zip) {
    $zipPath = "$OutDir.zip"
    if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
    Compress-Archive -Path $OutDir -DestinationPath $zipPath -Force
    Write-Host "Zip: $zipPath"
}
