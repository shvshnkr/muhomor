# Assemble portable Windows kit: exe + mihomo + launchers + Russian help.
# Run from repo root:  powershell -ExecutionPolicy Bypass -File .\scripts\pack-windows-kit.ps1
param(
    [string]$OutDir = "",
    [switch]$Zip,      # kept for compatibility; zip is on by default
    [switch]$NoZip,
    [switch]$SkipBuild
)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
if ($OutDir -eq "") {
    $OutDir = Join-Path $Root "dist\muhomor-kit"
}

& (Join-Path $Root "scripts\stop-muhomor.ps1")

$GccUcrt = "C:\msys64\ucrt64\bin"
$GccMingw = "C:\msys64\mingw64\bin"
if (Test-Path "$GccUcrt\gcc.exe") { $env:Path = "$GccUcrt;$env:Path" }
elseif (Test-Path "$GccMingw\gcc.exe") { $env:Path = "$GccMingw;$env:Path" }

Push-Location $Root
if (-not $SkipBuild) {
    Write-Host "Building muhomor.exe..."
    go build -o muhomor.exe ./cmd/muhomor
    if ($LASTEXITCODE -ne 0) { Pop-Location; throw "go build muhomor failed" }
    Write-Host "Building muhomor-gui.exe (Wails)..."
    & (Join-Path $Root "scripts\build-gui-wails.ps1") -OutFile (Join-Path $Root "muhomor-gui.exe") -NoPackKit
} else {
    Write-Host "Building muhomor.exe (daemon, -SkipBuild)..."
    go build -o muhomor.exe ./cmd/muhomor
    if ($LASTEXITCODE -ne 0) { Pop-Location; throw "go build muhomor failed" }
}
Pop-Location

$Mihomo = Join-Path $Root "bin\mihomo.exe"
if (-not (Test-Path $Mihomo)) {
    Write-Host "Fetching mihomo..."
    & (Join-Path $Root "scripts\fetch-mihomo-windows.ps1")
}

New-Item -ItemType Directory -Force -Path $OutDir, (Join-Path $OutDir "bin") | Out-Null

# Remove pre-data/ layout leftovers (cache/, run/, root *.db) so logs are only under data/.
foreach ($legacy in @("cache", "run", "muhomor.db", "muhomor.db-wal", "muhomor.db-shm")) {
    $p = Join-Path $OutDir $legacy
    if (Test-Path -LiteralPath $p) { Remove-Item -LiteralPath $p -Recurse -Force -ErrorAction SilentlyContinue }
}

Copy-Item (Join-Path $Root "muhomor.exe") $OutDir -Force
Copy-Item (Join-Path $Root "muhomor-gui.exe") $OutDir -Force
Copy-Item $Mihomo (Join-Path $OutDir "bin\mihomo.exe") -Force

$kitFiles = @(
    "СПРАВКА.txt",
    "Start-WG-mode3.bat", "Start-normal.bat", "Start-hidden.bat", "Test-Kit.bat",
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
    $sub = Join-Path $configDst "subscriptions.txt"
    $example = Join-Path $kitConfigSrc "subscriptions.example.txt"
    if ((Test-Path $sub) -and (Test-Path $example)) {
        $hasURL = $false
        foreach ($line in Get-Content -LiteralPath $sub -Encoding UTF8) {
            $t = $line.Trim()
            if ($t -ne "" -and -not $t.StartsWith("#")) { $hasURL = $true; break }
        }
        if (-not $hasURL) {
            Copy-Item -LiteralPath $example -Destination $sub -Force
        }
    }
}
$dataDir = Join-Path $OutDir "data"
if (Test-Path $dataDir) { Remove-Item -Recurse -Force $dataDir }
Write-Host "Initializing kit data (default settings DB)..."
Push-Location $Root
go run ./scripts/init-kit-data -o $dataDir
if ($LASTEXITCODE -ne 0) { Pop-Location; throw "init-kit-data failed" }
go run ./scripts/verify-kit-settings -db (Join-Path $dataDir "muhomor.db")
if ($LASTEXITCODE -ne 0) { Pop-Location; throw "verify-kit-settings failed" }
$geoDest = Join-Path $dataDir "run\mihomo\geoip.metadb"
Write-Host "Bundling geo database..."
go run ./scripts/fetch-geodata -o $geoDest
if ($LASTEXITCODE -ne 0) { Pop-Location; throw "fetch-geodata failed" }
Pop-Location
# Drop dev leftovers if data/ could not be fully removed (locked files)
foreach ($junk in @(
    "gui.lock", "muhomor.db-wal", "muhomor.db-shm",
    "cache\logs-history", "cache\pretest-last-path.txt", "cache\pretest-last.yaml",
    "run\config.yaml"
)) {
    $p = Join-Path $dataDir $junk
    if (Test-Path -LiteralPath $p) { Remove-Item -LiteralPath $p -Recurse -Force -ErrorAction SilentlyContinue }
}

Write-Host ""
Write-Host "Kit ready: $OutDir"
Get-ChildItem $OutDir, (Join-Path $OutDir "bin") | Format-Table Name, Length -AutoSize

if (-not $NoZip) {
    $zipPath = "$OutDir.zip"
    if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
    Write-Host "Creating zip..."
    Compress-Archive -Path $OutDir -DestinationPath $zipPath -Force
    $zipMb = [math]::Round((Get-Item $zipPath).Length / 1MB, 1)
    Write-Host "Zip: $zipPath ($zipMb MB)"
}
