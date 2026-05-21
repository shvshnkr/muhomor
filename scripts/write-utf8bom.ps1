param([Parameter(Mandatory)][string]$Path)
$c = Get-Content -LiteralPath $Path -Raw -Encoding UTF8
$utf8BOM = New-Object System.Text.UTF8Encoding $true
[System.IO.File]::WriteAllText((Resolve-Path -LiteralPath $Path).Path, $c, $utf8BOM)
