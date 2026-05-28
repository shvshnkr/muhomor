# Создать/обновить приватную копию mihomo (ветка Alpha) — репо muhomor-mihomo.
param(
    [string]$Repo = "shvshnkr/muhomor-mihomo",
    [string]$CloneDir = (Join-Path (Split-Path $PSScriptRoot -Parent) "..\muhomor-mihomo")
)

$gh = "${env:ProgramFiles}\GitHub CLI\gh.exe"
if (-not (Test-Path $gh)) { throw "GitHub CLI not found. Install: winget install GitHub.cli" }

& $gh auth status | Out-Null

if (-not (& $gh repo view $Repo 2>$null)) {
    & $gh repo create $Repo --private --description "Private mihomo (MetaCubeX Alpha) for muhomor" -y
}

if (-not (Test-Path $CloneDir)) {
    git clone --branch Alpha --single-branch "https://github.com/MetaCubeX/mihomo.git" $CloneDir
    Push-Location $CloneDir
    git remote rename origin upstream
    git remote add origin "https://github.com/$Repo.git"
    Pop-Location
}

Push-Location $CloneDir
git fetch upstream Alpha
git checkout Alpha
git push -u origin Alpha
Pop-Location

Write-Host "Done: https://github.com/$Repo (branch Alpha)"
Write-Host "Main muhomor app: https://github.com/shvshnkr/muhomor"
