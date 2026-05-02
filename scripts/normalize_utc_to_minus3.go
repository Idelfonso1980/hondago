package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

const migrationKey = "normalize_utc_to_minus3_20260422_v1"

func main() {
	dbPath := flag.String("db", "honda.sqlite", "caminho do banco sqlite")
	flag.Parse()

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("abrir banco: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping banco: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("iniciar transacao: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS maintenance_migrations (
  key TEXT PRIMARY KEY,
  applied_at DATETIME NOT NULL
)`); err != nil {
		log.Fatalf("criar maintenance_migrations: %v", err)
	}

	var already string
	err = tx.QueryRow(`SELECT key FROM maintenance_migrations WHERE key = ? LIMIT 1`, migrationKey).Scan(&already)
	if err == nil && already == migrationKey {
		fmt.Printf("Migracao %s ja aplicada. Nada a fazer.\n", migrationKey)
		if err := tx.Commit(); err != nil {
			log.Fatalf("commit: %v", err)
		}
		return
	}
	if err != nil && err != sql.ErrNoRows {
		log.Fatalf("verificar migration key: %v", err)
	}

	type upd struct {
		name string
		sql  string
	}

	updates := []upd{
		{
			name: "appuser.created_at",
			sql:  `UPDATE appuser SET created_at = datetime(created_at, '-3 hours') WHERE created_at IS NOT NULL AND TRIM(CAST(created_at AS TEXT)) <> ''`,
		},
		{
			name: "appuser.updated_at",
			sql:  `UPDATE appuser SET updated_at = datetime(updated_at, '-3 hours') WHERE updated_at IS NOT NULL AND TRIM(CAST(updated_at AS TEXT)) <> ''`,
		},
		{
			name: "appuser.last_login_at",
			sql:  `UPDATE appuser SET last_login_at = datetime(last_login_at, '-3 hours') WHERE last_login_at IS NOT NULL AND TRIM(CAST(last_login_at AS TEXT)) <> ''`,
		},
		{
			name: "appuser.locked_until",
			sql:  `UPDATE appuser SET locked_until = datetime(locked_until, '-3 hours') WHERE locked_until IS NOT NULL AND TRIM(CAST(locked_until AS TEXT)) <> ''`,
		},
		{
			name: "auth.last_request",
			sql:  `UPDATE auth SET last_request = datetime(last_request, '-3 hours') WHERE last_request IS NOT NULL AND TRIM(CAST(last_request AS TEXT)) <> ''`,
		},
		{
			name: "auth.cooldown_until",
			sql:  `UPDATE auth SET cooldown_until = datetime(cooldown_until, '-3 hours') WHERE cooldown_until IS NOT NULL AND TRIM(CAST(cooldown_until AS TEXT)) <> ''`,
		},
		{
			name: "auth.blocked_until",
			sql:  `UPDATE auth SET blocked_until = datetime(blocked_until, '-3 hours') WHERE blocked_until IS NOT NULL AND TRIM(CAST(blocked_until AS TEXT)) <> ''`,
		},
		{
			name: "idsgruposdisponiveis.created_at",
			sql:  `UPDATE idsgruposdisponiveis SET created_at = datetime(created_at, '-3 hours') WHERE created_at IS NOT NULL AND TRIM(CAST(created_at AS TEXT)) <> ''`,
		},
		{
			name: "cotasreservadas.created_at",
			sql:  `UPDATE cotasreservadas SET created_at = datetime(created_at, '-3 hours') WHERE created_at IS NOT NULL AND TRIM(CAST(created_at AS TEXT)) <> ''`,
		},
		{
			name: "solicitacoes.data_hora_atendimento",
			sql:  `UPDATE solicitacoes SET data_hora_atendimento = datetime(data_hora_atendimento, '-3 hours') WHERE data_hora_atendimento IS NOT NULL AND TRIM(CAST(data_hora_atendimento AS TEXT)) <> ''`,
		},
		{
			name: "solicitacoes.data_atendimento/hora_atendimento sync",
			sql: `UPDATE solicitacoes
SET data_atendimento = SUBSTR(TRIM(CAST(data_hora_atendimento AS TEXT)), 1, 10),
    hora_atendimento = SUBSTR(TRIM(CAST(data_hora_atendimento AS TEXT)), 12, 8)
WHERE data_hora_atendimento IS NOT NULL AND TRIM(CAST(data_hora_atendimento AS TEXT)) <> ''`,
		},
	}

	for _, u := range updates {
		res, err := tx.Exec(u.sql)
		if err != nil {
			log.Fatalf("executar %s: %v", u.name, err)
		}
		aff, _ := res.RowsAffected()
		fmt.Printf("%s -> %d linhas\n", u.name, aff)
	}

	_, err = tx.Exec(
		`INSERT INTO maintenance_migrations (key, applied_at) VALUES (?, ?)`,
		migrationKey,
		time.Now().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		log.Fatalf("registrar migration key: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit: %v", err)
	}
	fmt.Printf("Normalizacao concluida com sucesso em %s\n", *dbPath)
}

