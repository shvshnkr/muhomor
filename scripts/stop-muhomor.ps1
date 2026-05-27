# Stop muhomor GUI, daemon, and mihomo subprocess (unlock exe/db before pack or rebuild).
# Run from repo root:  powershell -ExecutionPolicy Bypass -File .\scripts\stop-muhomor.ps1
param(
    [switch]$Quiet
)

$ErrorActionPreference = "Continue"

function Write-StopMsg {
    param([string]$Text)
    if (-not $Quiet) { Write-Host $Text }
}

$names = @("muhomor-gui", "muhomor", "mihomo")
$stopped = 0

foreach ($name in $names) {
    $procs = Get-Process -Name $name -ErrorAction SilentlyContinue
    if (-not $procs) { continue }
    foreach ($p in $procs) {
        Write-StopMsg "Stopping $($p.ProcessName) (PID $($p.Id))..."
        Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue
        $stopped++
    }
}

if ($stopped -eq 0) {
    Write-StopMsg "No muhomor / mihomo processes running."
} else {
    # Brief wait so handles on exe/db are released before copy/build.
    Start-Sleep -Seconds 1
    $left = @()
    foreach ($name in $names) {
        if (Get-Process -Name $name -ErrorAction SilentlyContinue) { $left += $name }
    }
    if ($left.Count -gt 0) {
        Write-Warning "Still running: $($left -join ', '). Close manually or retry."
    } elseif (-not $Quiet) {
        Write-StopMsg "All stopped."
    }
}
