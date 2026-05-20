# Windows debug: proxy mode on port 2181
$ErrorActionPreference = "Continue"
$Root = Split-Path -Parent $PSScriptRoot
$Data = if ($env:MUHOMOR_DATA_DIR) { $env:MUHOMOR_DATA_DIR } else { Join-Path $env:LOCALAPPDATA "muhomor" }
$Port = if ($env:MUHOMOR_MIXED_PORT) { $env:MUHOMOR_MIXED_PORT } else { "2181" }
$LogDir = Join-Path $Data "cache"
$DaemonLog = Join-Path $LogDir "daemon-debug.log"
$DaemonErr = Join-Path $LogDir "daemon-debug.err.log"

New-Item -ItemType Directory -Force -Path $Data, $LogDir | Out-Null

$Muhomor = Join-Path $Root "muhomor.exe"
if (-not (Test-Path $Muhomor)) {
    Write-Host "Building muhomor..."
    Set-Location $Root
    go build -o muhomor.exe ./cmd/muhomor
}

Get-Process muhomor -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

Write-Host "Data dir: $Data"
Write-Host "Mixed port: $Port"

$BundledMihomo = Join-Path $Root "bin\mihomo.exe"
if (-not $env:MUHOMOR_MIHOMO_BIN -and (Test-Path $BundledMihomo)) {
    $env:MUHOMOR_MIHOMO_BIN = $BundledMihomo
    Write-Host "MUHOMOR_MIHOMO_BIN = $BundledMihomo"
}
if (-not $env:MUHOMOR_MIHOMO_BIN) {
    Write-Warning "Set MUHOMOR_MIHOMO_BIN or run scripts/fetch-mihomo-windows.ps1"
}

$argList = @("-dir", $Data, "--daemon", "--service-mode", "proxy", "--mixed-port", $Port)
$proc = Start-Process -FilePath $Muhomor -ArgumentList $argList -PassThru `
    -RedirectStandardOutput $DaemonLog -RedirectStandardError $DaemonErr -WindowStyle Hidden
Write-Host "Daemon PID $($proc.Id)"
Start-Sleep -Seconds 4

if ($env:MUHOMOR_VLESS_URI) {
    & $Muhomor -dir $Data --import-uri $env:MUHOMOR_VLESS_URI
}

Write-Host "Profiles count:" (& $Muhomor -dir $Data --profiles 2>&1 | Measure-Object -Line).Lines
Write-Host "ctl start (may take minutes with many profiles)..."
& $Muhomor -dir $Data --ctl start
Start-Sleep -Seconds 8
& $Muhomor -dir $Data --ctl status

$proxy = "http://127.0.0.1:$Port"
$cfgPath = Join-Path $Data "run\config.yaml"
if (Test-Path $cfgPath) {
    $cfg = Get-Content $cfgPath -Raw
    if ($cfg -match 'authentication:\s*\r?\n\s*-\s*"([^"]+)"') {
        $proxy = "http://$($matches[1])@127.0.0.1:$Port"
        Write-Host "Proxy auth from config.yaml"
    }
}
Write-Host "IP direct:"
curl.exe -s --max-time 15 https://api.ipify.org
Write-Host ""
Write-Host "IP via proxy:"
curl.exe -s --max-time 30 -x $proxy https://api.ipify.org
Write-Host ""

Write-Host "Daemon stdout tail:"
if (Test-Path $DaemonLog) { Get-Content $DaemonLog -Tail 25 }
Write-Host "Daemon stderr tail:"
if (Test-Path $DaemonErr) { Get-Content $DaemonErr -Tail 25 }

try {
    Invoke-RestMethod -Uri "http://127.0.0.1:8751/v1/service/status" | ConvertTo-Json
} catch {
    Write-Host "REST status failed: $_"
}

Write-Host "Stop daemon: Stop-Process -Id $($proc.Id) -Force"
