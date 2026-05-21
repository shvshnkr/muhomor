# Dev launcher: WG-over-WL preset 3. Use .bat in kit if PS1 is blocked.
#   powershell -ExecutionPolicy Bypass -File .\scripts\launch-wg-mode3.ps1
$Root = Split-Path -Parent $PSScriptRoot
$Bat = Join-Path $Root "dist\muhomor-kit\Запуск-WG-режим3.bat"
if (Test-Path $Bat) {
    & cmd /c "`"$Bat`""
    exit $LASTEXITCODE
}
$env:MUHOMOR_MIHOMO_BIN = Join-Path $Root "bin\mihomo.exe"
$Data = if ($env:MUHOMOR_DATA_DIR) { $env:MUHOMOR_DATA_DIR } else { Join-Path $env:LOCALAPPDATA "muhomor" }
$Gui = Join-Path $Root "muhomor-gui.exe"
if (-not (Test-Path $Gui)) { throw "Build GUI first: go build -tags cgo -o muhomor-gui.exe ./cmd/muhomor-gui" }
Start-Process -FilePath $Gui -ArgumentList @(
    "-dir", $Data,
    "--service-mode", "vpn",
    "--mixed-port", "2181",
    "--route-quick-profile", "3"
) -WorkingDirectory $Root
