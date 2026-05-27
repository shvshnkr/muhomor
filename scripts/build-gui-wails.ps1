# Build muhomor-gui with Wails (Windows amd64).
# Requires: Node 18+, Wails CLI (go install github.com/wailsapp/wails/v2/cmd/wails@latest)
param(
    [string]$OutFile = "muhomor-gui.exe",
    [switch]$NoPackKit
)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$env:Path = "$env:USERPROFILE\go\bin;$env:Path"

Push-Location $Root
try {
    Write-Host "npm install (frontend)..."
    Push-Location (Join-Path $Root "frontend")
    npm ci 2>$null
    if ($LASTEXITCODE -ne 0) { npm install }
    npm run build
    if ($LASTEXITCODE -ne 0) { throw "frontend build failed" }
    Pop-Location

    # Wails -o must be a filename only (full paths break output dir on Windows).
    $outLeaf = Split-Path -Leaf $OutFile
    if (-not $outLeaf) { $outLeaf = "muhomor-gui.exe" }
    Write-Host "wails build -> build\bin\$outLeaf"
    wails build -platform windows/amd64 -o $outLeaf
    if ($LASTEXITCODE -ne 0) { throw "wails build failed" }
    $built = Join-Path $Root "build\bin\$outLeaf"
    if (-not (Test-Path $built)) {
        $built = Join-Path $Root "build\bin\muhomor-gui.exe"
    }
    $dest = if ([System.IO.Path]::IsPathRooted($OutFile)) { $OutFile } else { Join-Path $Root $OutFile }
    if (Test-Path $built) {
        Copy-Item $built $dest -Force
        Write-Host "Copied to $dest"
    } else {
        throw "wails output not found: $built"
    }

    if (-not $NoPackKit) {
        Write-Host "Packing Windows kit (-SkipBuild, GUI just built)..."
        & (Join-Path $Root "scripts\pack-windows-kit.ps1") -SkipBuild
    }
}
finally {
    Pop-Location
}
