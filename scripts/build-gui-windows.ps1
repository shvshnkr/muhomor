# Build muhomor-gui.exe without a console window (Windows GUI subsystem).
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Out = if ($args[0]) { $args[0] } else { Join-Path $Root "muhomor-gui.exe" }

$GccUcrt = "C:\msys64\ucrt64\bin"
$GccMingw = "C:\msys64\mingw64\bin"
if (Test-Path "$GccUcrt\gcc.exe") { $env:Path = "$GccUcrt;$env:Path" }
elseif (Test-Path "$GccMingw\gcc.exe") { $env:Path = "$GccMingw;$env:Path" }

$env:CGO_ENABLED = "1"
Push-Location $Root
try {
    go build -tags cgo -ldflags "-H windowsgui" -o $Out ./cmd/muhomor-gui
    Write-Host "Built: $Out (subsystem=windows, no console)"
} finally {
    Pop-Location
}
