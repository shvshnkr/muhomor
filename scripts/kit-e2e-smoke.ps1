# Kit e2e smoke: daemon API (default), optional hidden Wails GUI.
# Run from repo root:
#   powershell -ExecutionPolicy Bypass -File .\scripts\kit-e2e-smoke.ps1
#   powershell -File .\scripts\kit-e2e-smoke.ps1 -UILauncher gui
# Tier B (full connect): $env:MUHOMOR_TEST_IMPORT_URI = 'vless://...'
param(
    [string]$KitDir = "",
    [ValidateSet("daemon", "gui", "pseudo", "all")]
    [string]$UILauncher = "daemon",
    [switch]$SkipPack,
    [switch]$MinimalPack,
    [int]$ApiTimeoutSec = 60,
    [int]$ConnectPollSec = 180
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
if ($KitDir -eq "") {
    $KitDir = Join-Path $Root "dist\muhomor-kit"
}

function Write-Smoke([string]$Msg) { Write-Host "[kit-e2e] $Msg" }

function Stop-KitProcesses {
    & (Join-Path $Root "scripts\stop-muhomor.ps1") -Quiet
}

function Ensure-MinimalKit {
    Write-Smoke "minimal kit pack (daemon smoke, no Wails build)"
    New-Item -ItemType Directory -Force -Path $KitDir, (Join-Path $KitDir "bin") | Out-Null
    Push-Location $Root
    go build -o (Join-Path $KitDir "muhomor.exe") ./cmd/muhomor
    if ($LASTEXITCODE -ne 0) { Pop-Location; throw "go build muhomor failed" }
    Pop-Location
    $mihomo = Join-Path $Root "bin\mihomo.exe"
    if (-not (Test-Path $mihomo)) {
        & (Join-Path $Root "scripts\fetch-mihomo-windows.ps1")
        $mihomo = Join-Path $Root "bin\mihomo.exe"
    }
    Copy-Item $mihomo (Join-Path $KitDir "bin\mihomo.exe") -Force
    New-Item -ItemType File -Force -Path (Join-Path $KitDir ".muhomor-portable") | Out-Null
    Set-Content -Path (Join-Path $KitDir "VERSION.txt") -Value "muhomor-kit`ne2e-minimal" -Encoding UTF8
    $dataDir = Join-Path $KitDir "data"
    if (Test-Path $dataDir) { Remove-Item -Recurse -Force $dataDir }
    Push-Location $Root
    go run ./scripts/init-kit-data -o $dataDir
    if ($LASTEXITCODE -ne 0) { Pop-Location; throw "init-kit-data failed" }
    Pop-Location
    if ($UILauncher -eq "gui" -or $UILauncher -eq "all") {
        $gui = Join-Path $KitDir "muhomor-gui.exe"
        if (-not (Test-Path $gui)) {
            $built = Join-Path $Root "muhomor-gui.exe"
            if (Test-Path $built) {
                Copy-Item $built $gui -Force
            } else {
                throw "muhomor-gui.exe required for -UILauncher gui/all; build GUI or use full pack"
            }
        }
    }
}

function Ensure-Kit {
    if ($SkipPack) {
        if (-not (Test-Path (Join-Path $KitDir "muhomor.exe"))) {
            throw "Kit not found at $KitDir (drop -SkipPack, use -MinimalPack, or pack first)"
        }
        return
    }
    if ($MinimalPack) {
        Ensure-MinimalKit
        return
    }
    Write-Smoke "pack-windows-kit.ps1 -NoZip"
    & (Join-Path $Root "scripts\pack-windows-kit.ps1") -NoZip -OutDir $KitDir
}

function Set-KitEnv {
    $script:KitRoot = (Resolve-Path -LiteralPath $KitDir).Path
    $script:DataDir = Join-Path $KitRoot "data"
    $script:CacheDir = Join-Path $DataDir "cache"
    $env:MUHOMOR_MIHOMO_BIN = Join-Path $KitRoot "bin\mihomo.exe"
    $env:MUHOMOR_DATA_DIR = $DataDir
    if (-not (Test-Path $DataDir)) { New-Item -ItemType Directory -Force -Path $DataDir | Out-Null }
}

function Start-DaemonProcess {
    $exe = Join-Path $KitRoot "muhomor.exe"
    $args = @("--daemon", "-dir", $DataDir, "--service-mode", "proxy", "--mixed-port", "2181")
    Write-Smoke "Start-Process $exe $($args -join ' ')"
    $script:LauncherProc = Start-Process -FilePath $exe -ArgumentList $args -WorkingDirectory $KitRoot -PassThru
}

function Start-GuiProcess {
    $exe = Join-Path $KitRoot "muhomor-gui.exe"
    $args = @("-dir", $DataDir, "-start-hidden", "--service-mode", "proxy", "--mixed-port", "2181")
    Write-Smoke "Start-Process $exe $($args -join ' ')"
    $script:LauncherProc = Start-Process -FilePath $exe -ArgumentList $args -WorkingDirectory $KitRoot -PassThru
}

function Wait-DaemonApi {
    $deadline = (Get-Date).AddSeconds($ApiTimeoutSec)
    $uri = "http://127.0.0.1:8751/v1/service/status"
    while ((Get-Date) -lt $deadline) {
        try {
            $null = Invoke-RestMethod -Uri $uri -Method Get -TimeoutSec 5
            Write-Smoke "daemon API ready"
            return
        } catch {
            Start-Sleep -Seconds 1
        }
    }
    throw "daemon API not ready after ${ApiTimeoutSec}s"
}

function Invoke-DaemonApi {
    param(
        [string]$Method,
        [string]$Path,
        [object]$Body = $null,
        [int]$TimeoutSec = 30
    )
    $uri = "http://127.0.0.1:8751$Path"
    $params = @{
        Uri         = $uri
        Method      = $Method
        TimeoutSec  = $TimeoutSec
        ErrorAction = "Stop"
    }
    if ($null -ne $Body) {
        $params.ContentType = "application/json"
        $params.Body = ($Body | ConvertTo-Json -Compress)
    }
    return Invoke-RestMethod @params
}

function Test-ExpectedStartFailure([string]$Err) {
    if ([string]::IsNullOrWhiteSpace($Err)) { return $false }
    $patterns = @(
        "no profiles",
        "bootstrap",
        "import-uri",
        "profile"
    )
    foreach ($p in $patterns) {
        if ($Err -match $p) { return $true }
    }
    return $false
}

function Invoke-TierA {
    param([string]$Label)
    Write-Smoke "Tier A ($Label): start -> status -> stop"
    $startErr = $null
    try {
        $null = Invoke-DaemonApi -Method POST -Path "/v1/service/start" -TimeoutSec 120
    } catch {
        $startErr = $_.Exception.Message
        if ($_.ErrorDetails.Message) { $startErr = $_.ErrorDetails.Message }
    }
    if ($startErr) {
        if (Test-ExpectedStartFailure $startErr) {
            Write-Smoke "start returned expected error (no profiles): $startErr"
        } else {
            throw "unexpected start error: $startErr"
        }
    }

    $st = Invoke-DaemonApi -Method GET -Path "/v1/service/status"
    Write-Smoke "status state=$($st.state) connected=$($st.connected)"

    $ctl = Join-Path $CacheDir "desktop-control-status.txt"
    if (-not (Test-Path $ctl)) {
        throw "missing desktop-control-status.txt"
    }
    $ctlText = Get-Content -LiteralPath $ctl -Raw -Encoding UTF8
    if ($ctlText -notmatch "state=") {
        throw "desktop-control-status.txt missing state="
    }

    $null = Invoke-DaemonApi -Method POST -Path "/v1/service/stop" -TimeoutSec 120
    $st2 = Invoke-DaemonApi -Method GET -Path "/v1/service/status"
    if ($st2.connected -eq $true) {
        throw "still connected after stop"
    }

    $log = Join-Path $CacheDir "daemon-debug.err.log"
    if (Test-Path $log) {
        $hits = Select-String -Path $log -Pattern "daemon not running" -SimpleMatch -ErrorAction SilentlyContinue
        if ($hits) {
            throw "false 'daemon not running' in daemon-debug.err.log ($($hits.Count) hits)"
        }
    }
    Write-Smoke "Tier A OK ($Label)"
}

function Import-TestProfile {
    $uri = $env:MUHOMOR_TEST_IMPORT_URI
    if ([string]::IsNullOrWhiteSpace($uri)) { return }
    Write-Smoke "Tier B: import profile"
    $null = Invoke-DaemonApi -Method POST -Path "/v1/profiles/import" -Body @{ uri = $uri }
}

function Invoke-TierB {
    param([string]$Label)
    $uri = $env:MUHOMOR_TEST_IMPORT_URI
    if ([string]::IsNullOrWhiteSpace($uri)) {
        Write-Smoke "Tier B skipped (MUHOMOR_TEST_IMPORT_URI not set)"
        return
    }
    Write-Smoke "Tier B ($Label): connect + ping + stop"
    $null = Invoke-DaemonApi -Method POST -Path "/v1/service/start" -TimeoutSec $ConnectPollSec

    $deadline = (Get-Date).AddSeconds($ConnectPollSec)
    $connected = $false
    while ((Get-Date) -lt $deadline) {
        $st = Invoke-DaemonApi -Method GET -Path "/v1/service/status"
        if ($st.connected -eq $true) {
            $connected = $true
            break
        }
        Start-Sleep -Seconds 2
    }
    if (-not $connected) {
        throw "Tier B: connected=true not reached within ${ConnectPollSec}s"
    }

    try {
        $null = Invoke-DaemonApi -Method POST -Path "/v1/service/ping" -TimeoutSec 60
    } catch {
        Write-Smoke "ping warning: $($_.Exception.Message)"
    }

    $null = Invoke-DaemonApi -Method POST -Path "/v1/service/stop" -TimeoutSec 120
    Write-Smoke "Tier B OK ($Label)"
}

function Stop-DaemonApi {
    try {
        $null = Invoke-DaemonApi -Method POST -Path "/v1/daemon/shutdown" -TimeoutSec 15
    } catch {
        Write-Smoke "shutdown API: $($_.Exception.Message)"
    }
}

function Invoke-LauncherSmoke {
    param(
        [ValidateSet("daemon", "gui", "pseudo")]
        [string]$Mode,
        [switch]$SkipTierB
    )
    if ($Mode -eq "pseudo") {
        Write-Smoke "pseudo launcher = daemon API parity (no TTY)"
        $Mode = "daemon"
    }
    Stop-KitProcesses
    Set-KitEnv
    if ($Mode -eq "gui") {
        Start-GuiProcess
    } else {
        Start-DaemonProcess
    }
    Wait-DaemonApi
    Import-TestProfile
    Invoke-TierA -Label $Mode
    if (-not $SkipTierB) {
        Invoke-TierB -Label $Mode
    }
    Stop-DaemonApi
    Stop-KitProcesses
    if ($script:LauncherProc -and -not $script:LauncherProc.HasExited) {
        Stop-Process -Id $script:LauncherProc.Id -Force -ErrorAction SilentlyContinue
    }
}

# --- main ---
Write-Smoke "UILauncher=$UILauncher KitDir=$KitDir"
Stop-KitProcesses
Ensure-Kit

switch ($UILauncher) {
    "daemon" { Invoke-LauncherSmoke -Mode daemon }
    "gui"    { Invoke-LauncherSmoke -Mode gui }
    "pseudo" { Invoke-LauncherSmoke -Mode pseudo }
    "all" {
        Invoke-LauncherSmoke -Mode daemon
        Invoke-LauncherSmoke -Mode gui -SkipTierB
    }
}

Write-Smoke "PASS"
