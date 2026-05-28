# Download official mihomo Windows amd64 from MetaCubeX/mihomo releases.
param(
    [string]$Version = "v1.19.25",
    [string]$OutDir = (Join-Path (Split-Path $PSScriptRoot -Parent) "bin")
)
$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$zip = Join-Path $OutDir "mihomo.zip"
$asset = "mihomo-windows-amd64-$Version.zip"
$url = "https://github.com/MetaCubeX/mihomo/releases/download/$Version/$asset"

Write-Host "Downloading $asset ..."

$downloaded = $false
if ($env:GH_TOKEN -or (Get-Command gh -ErrorAction SilentlyContinue)) {
    $Gh = "C:\Program Files\GitHub CLI\gh.exe"
    if (-not (Test-Path $Gh)) { $Gh = "gh" }
    try {
        & $Gh release download $Version -R MetaCubeX/mihomo -p $asset -O $zip
        if (Test-Path $zip) { $downloaded = $true }
    } catch {
        Write-Host "gh release download failed: $_"
    }
}

if (-not $downloaded) {
    Write-Host "Fallback: $url"
    Invoke-WebRequest -Uri $url -OutFile $zip -UseBasicParsing
}

Expand-Archive -Path $zip -DestinationPath $OutDir -Force
Remove-Item $zip -Force
$extracted = Get-ChildItem $OutDir -Filter "mihomo*.exe" | Where-Object { $_.Name -ne "mihomo.exe" } | Select-Object -First 1
if ($extracted) {
    Move-Item $extracted.FullName (Join-Path $OutDir "mihomo.exe") -Force
}
$exe = Join-Path $OutDir "mihomo.exe"
if (-not (Test-Path $exe)) {
    throw "mihomo.exe not found under $OutDir after extract"
}
& $exe -v
Write-Host "Installed: $exe"
Write-Host "Use: `$env:MUHOMOR_MIHOMO_BIN = '$exe'"
