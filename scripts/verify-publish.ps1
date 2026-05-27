# Pre-push publish hygiene checks. Run from repo root.
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Push-Location $Root

Write-Host "go test ./... (CGO_ENABLED=0)"
$env:CGO_ENABLED = "0"
go test ./...
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }

Write-Host "go vet ./..."
go vet ./...
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }

$patterns = @(
    "C:\\Users\\",
    "debug-5ee762",
    "agentDebugLog",
    "agent_debug"
)
foreach ($pat in $patterns) {
    $hits = git grep -n $pat -- ':!scripts/verify-publish.ps1' ':!scripts/verify-publish.sh' ':!docs/PUBLISHING.md' 2>$null
    if ($LASTEXITCODE -eq 0 -and $hits) {
        Write-Error ("Forbidden pattern in git tree: " + $pat + [Environment]::NewLine + $hits)
    }
}

$untracked = git status --porcelain
if ($untracked -match '^\?\? dist/' -or $untracked -match '\.db') {
    Write-Warning "Untracked dist/ or *.db present - do not add them to the commit."
}

Write-Host "verify-publish: OK"
Pop-Location
