# MSYS2 gcc + muhomor bin for local Windows dev (run in PowerShell).
$Root = Split-Path -Parent $PSScriptRoot
$Mihomo = Join-Path $Root "bin\mihomo.exe"
$GccUcrt = "C:\msys64\ucrt64\bin"
$GccMingw = "C:\msys64\mingw64\bin"

if (Test-Path $GccUcrt\gcc.exe) {
    $env:Path = "$GccUcrt;$env:Path"
    Write-Host "PATH += $GccUcrt"
} elseif (Test-Path $GccMingw\gcc.exe) {
    $env:Path = "$GccMingw;$env:Path"
    Write-Host "PATH += $GccMingw"
} else {
    Write-Warning "gcc not found. Install MSYS2: pacman -S mingw-w64-ucrt-x86_64-gcc"
}

if (-not (Test-Path $Mihomo)) {
    Write-Host "mihomo missing — run: .\scripts\fetch-mihomo-windows.ps1"
} else {
    $env:MUHOMOR_MIHOMO_BIN = $Mihomo
    Write-Host "MUHOMOR_MIHOMO_BIN = $Mihomo"
}

$env:CGO_ENABLED = "1"
Write-Host "CGO_ENABLED=1"
Write-Host ""
Write-Host "Build GUI:"
Write-Host "  go build -o muhomor-gui.exe ./cmd/muhomor-gui"
Write-Host "Debug proxy :2181:"
Write-Host "  `$env:MUHOMOR_MIXED_PORT='2181'; .\scripts\debug-windows.ps1"
