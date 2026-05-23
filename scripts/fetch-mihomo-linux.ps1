# Download official mihomo Linux amd64 from MetaCubeX/mihomo releases.
param(
    [string]$Version = "v1.19.25",
    [ValidateSet("amd64", "arm64")]
    [string]$Arch = "amd64",
    [string]$OutDir = (Join-Path (Split-Path $PSScriptRoot -Parent) "bin")
)
$ErrorActionPreference = "Stop"
$Gh = "C:\Program Files\GitHub CLI\gh.exe"
if (-not (Test-Path $Gh)) { $Gh = "gh" }

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
if ($Arch -eq "arm64") {
    $asset = "mihomo-linux-arm64-$Version.gz"
} else {
    $asset = "mihomo-linux-amd64-compatible-$Version.gz"
}
$gz = Join-Path $OutDir "mihomo-linux.gz"
$bin = Join-Path $OutDir "mihomo"
Write-Host "Downloading $asset ..."
& $Gh release download $Version -R MetaCubeX/mihomo -p $asset -O $gz
# Release .gz is a single executable, not tar.gz
$in = [System.IO.File]::OpenRead($gz)
try {
    $gzip = New-Object System.IO.Compression.GzipStream($in, [IO.Compression.CompressionMode]::Decompress)
    try {
        if (Test-Path $bin) { Remove-Item $bin -Force }
        $out = [System.IO.File]::Create($bin)
        try { $gzip.CopyTo($out) } finally { $out.Close() }
    } finally { $gzip.Close() }
} finally { $in.Close() }
Remove-Item $gz -Force
if (-not (Test-Path $bin)) { throw "mihomo binary not found in $OutDir" }
& $bin -v
Write-Host "Installed: $bin"
