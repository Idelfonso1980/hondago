param(
  [string]$Root = "."
)

$ErrorActionPreference = "Stop"

Set-Location $Root

$include = @("*.go", "*.md", "*.ini", "*.json", "*.yaml", "*.yml", "*.sh", "*.ps1", "*.command", "*.txt")
$excludeDirs = @("dist", "log", ".git", ".idea", ".vscode")
$excludeFilePattern = @("*.bak", "*.tmp", "*.old", "*.orig", "*.exe", "*.dll", "*.png", "*.jpg", "*.jpeg", "*.gif", "*.svg", "*.pdf")

function Is-ExcludedFile([System.IO.FileInfo]$f) {
  if ($f.Name -ieq "check_encoding.ps1") { return $true }
  foreach ($d in $excludeDirs) {
    if ($f.FullName -like "*\\$d\\*") { return $true }
  }
  foreach ($p in $excludeFilePattern) {
    if ($f.Name -like $p) { return $true }
  }
  return $false
}

function Is-ValidUtf8([byte[]]$bytes) {
  try {
    [Text.UTF8Encoding]::new($false, $true).GetString($bytes) | Out-Null
    return $true
  } catch {
    return $false
  }
}

$bad = New-Object System.Collections.Generic.List[string]
$mojibake = New-Object System.Collections.Generic.List[string]
$rx = [regex]"(`u00C3`u0192|`u00C3`u201A|`u00C3`u00A2`u20AC|`u00C2`u00A0|`u0192)"

$files = Get-ChildItem -Recurse -File -Include $include | Where-Object { -not (Is-ExcludedFile $_) }
foreach ($f in $files) {
  $bytes = [IO.File]::ReadAllBytes($f.FullName)
  if (-not (Is-ValidUtf8 $bytes)) {
    $bad.Add($f.FullName)
    continue
  }
  $txt = [Text.UTF8Encoding]::new($false, $true).GetString($bytes)
  if ($rx.IsMatch($txt)) {
    $mojibake.Add($f.FullName)
  }
}

if ($bad.Count -gt 0 -or $mojibake.Count -gt 0) {
  Write-Host "Encoding check failed." -ForegroundColor Red
  if ($bad.Count -gt 0) {
    Write-Host "`nInvalid UTF-8 files:" -ForegroundColor Yellow
    $bad | Sort-Object -Unique | ForEach-Object { Write-Host " - $_" }
  }
  if ($mojibake.Count -gt 0) {
    Write-Host "`nPotential mojibake files:" -ForegroundColor Yellow
    $mojibake | Sort-Object -Unique | ForEach-Object { Write-Host " - $_" }
  }
  exit 1
}

Write-Host "Encoding check passed." -ForegroundColor Green
