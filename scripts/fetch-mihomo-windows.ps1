# Download official mihomo Windows amd64 from MetaCubeX/mihomo releases.
param(
    [string]$Version = "v1.19.25",
    [string]$OutDir = (Join-Path (Split-Path $PSScriptRoot -Parent) "bin")
)
$ErrorActionPreference = "Stop"
$Gh = "C:\Program Files\GitHub CLI\gh.exe"
if (-not (Test-Path $Gh)) { $Gh = "gh" }

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$zip = Join-Path $OutDir "mihomo.zip"
$pattern = "mihomo-windows-amd64-$Version.zip" -replace '^v','v'
# asset name uses version with v prefix
$asset = "mihomo-windows-amd64-$Version.zip"

Write-Host "Downloading $asset ..."
& $Gh release download $Version -R MetaCubeX/mihomo -p $asset -O $zip
Expand-Archive -Path $zip -DestinationPath $OutDir -Force
Remove-Item $zip -Force
$extracted = Get-ChildItem $OutDir -Filter "mihomo*.exe" | Where-Object { $_.Name -ne "mihomo.exe" } | Select-Object -First 1
if ($extracted) {
    Move-Item $extracted.FullName (Join-Path $OutDir "mihomo.exe") -Force
}
$exe = Join-Path $OutDir "mihomo.exe"
& $exe -v
Write-Host "Installed: $exe"
Write-Host "Use: `$env:MUHOMOR_MIHOMO_BIN = '$exe'"
