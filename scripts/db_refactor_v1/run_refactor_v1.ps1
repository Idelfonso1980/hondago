param(
  [ValidateSet("dry-run","apply","rollback")]
  [string]$Mode = "dry-run",
  [string]$DbPath = "honda.sqlite"
)

$ErrorActionPreference = "Stop"

function Require-SqliteCli {
  $sqlite = Get-Command sqlite3 -ErrorAction SilentlyContinue
  return [bool]$sqlite
}

function Exec-SqlFile([string]$db, [string]$sqlFile) {
  if (-not (Test-Path $sqlFile)) {
    throw "Arquivo SQL nao encontrado: $sqlFile"
  }
  Write-Host ">> Executando: $sqlFile"
  if ($script:UseSqliteCli) {
    & sqlite3 $db ".read $sqlFile"
  } else {
    go run ./scripts/db_refactor_v1/sql_runner.go $db $sqlFile
  }
}

$root = Split-Path -Parent $MyInvocation.MyCommand.Path

if (-not (Test-Path $DbPath)) {
  throw "Banco nao encontrado: $DbPath"
}

$script:UseSqliteCli = Require-SqliteCli
if (-not $script:UseSqliteCli) {
  Write-Host "sqlite3 CLI nao encontrado. Usando runner Go (modernc.org/sqlite)." -ForegroundColor Yellow
}

$createSql   = Join-Path $root "01_create_prof_schema.sql"
$backfillSql = Join-Path $root "02_backfill_prof_schema.sql"
$validateSql = Join-Path $root "03_validate_prof_schema.sql"
$rollbackSql = Join-Path $root "05_rollback_prof_schema.sql"

switch ($Mode) {
  "dry-run" {
    Write-Host "Modo DRY-RUN: nenhuma alteracao sera aplicada."
    Write-Host "Banco: $DbPath"
    Write-Host "Scripts detectados:"
    Write-Host " - $createSql"
    Write-Host " - $backfillSql"
    Write-Host " - $validateSql"
    Write-Host ""
    if ($script:UseSqliteCli) {
      & sqlite3 $DbPath "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;"
    } else {
      Write-Host "Dry-run concluido (runner Go disponível)." -ForegroundColor Green
    }
    break
  }
  "apply" {
    Write-Host "Modo APPLY: criar schema v2 + backfill + validate."
    $stamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $backup = "$DbPath.bak_$stamp"
    Copy-Item $DbPath $backup -Force
    Write-Host "Backup criado: $backup"

    Exec-SqlFile $DbPath $createSql
    Exec-SqlFile $DbPath $backfillSql
    Exec-SqlFile $DbPath $validateSql

    Write-Host "APPLY concluido."
    break
  }
  "rollback" {
    Write-Host "Modo ROLLBACK: remover tabelas *_v2 desta refatoracao."
    Exec-SqlFile $DbPath $rollbackSql
    Write-Host "ROLLBACK concluido."
    break
  }
}
