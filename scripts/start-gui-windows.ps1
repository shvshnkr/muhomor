# Start muhomor-gui on Windows (proxy test on port 2181 by default).
$Root = Split-Path -Parent $PSScriptRoot
. (Join-Path $PSScriptRoot "setup-windows-dev.ps1")

$Data = if ($env:MUHOMOR_DATA_DIR) { $env:MUHOMOR_DATA_DIR } else { Join-Path $env:LOCALAPPDATA "muhomor" }
$Port = if ($env:MUHOMOR_MIXED_PORT) { $env:MUHOMOR_MIXED_PORT } else { "2181" }
$Mode = if ($env:MUHOMOR_SERVICE_MODE) { $env:MUHOMOR_SERVICE_MODE } else { "proxy" }

$Gui = Join-Path $Root "muhomor-gui.exe"
if (-not (Test-Path $Gui)) {
    & (Join-Path $PSScriptRoot "build-gui-windows.ps1") $Gui
}

Write-Host "Starting GUI: mode=$Mode port=$Port data=$Data"
Write-Host "Connect = simple mode (selector picks best from ALL enabled profiles, not one random URI)"
Write-Host "Tray: Proxy / VPN switches service_mode + reload"

Start-Process -FilePath $Gui -ArgumentList @(
    "-dir", $Data,
    "--service-mode", $Mode,
    "--mixed-port", $Port
) -WorkingDirectory $Root
