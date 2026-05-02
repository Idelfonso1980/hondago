param(
  [string]$DbPath = ".\honda.db",
  [string]$OutDir = ".\tmp\sqlite_csv"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $DbPath)) {
  throw "Banco SQLite não encontrado em: $DbPath"
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "Go não encontrado no PATH. Instale o Go para usar este exportador."
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$goCode = @'
package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var tables = []string{
	"roles",
	"users",
	"role_permissions",
	"api_accounts",
	"vendor_identity_map",
	"requests",
	"reservations",
	"available_group_ids",
	"models",
	"products",
	"active_groups",
	"holidays",
	"assemblies",
	"manual_notifications",
	"audit_log",
	"app_sessions",
}

func quoteIdent(v string) string {
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}

func tableColumns(ctx context.Context, db *sql.DB, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+quoteIdent(table)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case []byte:
		return string(t)
	case time.Time:
		return t.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprint(t)
	}
}

func exportTable(ctx context.Context, db *sql.DB, outDir, table string) error {
	cols, err := tableColumns(ctx, db, table)
	if err != nil {
		return fmt.Errorf("colunas %s: %w", table, err)
	}
	if len(cols) == 0 {
		return nil
	}

	outPath := filepath.Join(outDir, table+".csv")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(cols); err != nil {
		return err
	}

	colList := make([]string, 0, len(cols))
	for _, c := range cols {
		colList = append(colList, quoteIdent(c))
	}
	q := "SELECT " + strings.Join(colList, ", ") + " FROM " + quoteIdent(table)
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return fmt.Errorf("select %s: %w", table, err)
	}
	defer rows.Close()

	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		record := make([]string, len(cols))
		for i := range vals {
			record[i] = anyToString(vals[i])
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "uso: exporter <sqlite.db> <out_dir>")
		os.Exit(2)
	}
	dbPath := os.Args[1]
	outDir := os.Args[2]

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	for _, t := range tables {
		if err := exportTable(ctx, db, outDir, t); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao exportar %s: %v\n", t, err)
			os.Exit(1)
		}
		fmt.Printf("ok: %s.csv\n", t)
	}
}
'@

$tmpGo = Join-Path $env:TEMP "hondago_sqlite_exporter.go"
Set-Content -LiteralPath $tmpGo -Value $goCode -Encoding UTF8

try {
  & go run $tmpGo $DbPath $OutDir
  Write-Host ""
  Write-Host "Export concluído em: $OutDir"
  Write-Host "Próximo passo (psql):"
  Write-Host "\set csv_dir '$((Resolve-Path $OutDir).Path -replace '\\','/')'"
  Write-Host "\i docs/sql/postgres/003_migrate_from_sqlite.sql"
}
finally {
  Remove-Item -LiteralPath $tmpGo -ErrorAction SilentlyContinue
}
