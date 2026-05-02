# SQLite -> Postgres Compat Checklist

Status atual: `em preparacao` (DATABASE_URL aceito, fallback SQLite ativo).

## 1) Conexao e bootstrap

1. `Concluido` Suporte a `HONDAGO_DATABASE_URL` no carregamento de configuracao.
2. `Concluido` Abertura de conexao por URL Postgres (`pgx`) com fallback SQLite.
3. `Pendente` Fluxo de `EnsureLegacySchema` separado por dialeto.

## 2) SQL dependente de SQLite (mapa de ajuste)

1. `Pendente` `INSERT OR IGNORE` -> trocar por `INSERT ... ON CONFLICT DO NOTHING`.
Arquivos: `internal/db/schema.go`.

2. `Pendente` `INSERT OR REPLACE` -> trocar por `INSERT ... ON CONFLICT (...) DO UPDATE`.
Arquivos: `internal/db/db.go` (`app_sessions`).

3. `Pendente` `sqlite_master` -> trocar por `information_schema` / `pg_catalog`.
Arquivos: `internal/db/schema.go`, `internal/db/db.go`.

4. `Pendente` `PRAGMA table_info` -> trocar por `information_schema.columns`.
Arquivos: `internal/db/schema.go`, `internal/db/db.go`.

5. `Pendente` `DELETE FROM sqlite_sequence` -> remover e usar `setval` quando necessario.
Arquivos: `internal/db/schema.go`.

6. `Pendente` Placeholders `?` -> adaptar para `$1..$n` no driver Postgres.
Arquivos: `internal/db/db.go` (queries de CRUD e relatorios).

7. `Pendente` Backup/restore SQL com sintaxe SQLite (`PRAGMA`, quoting) -> criar versao Postgres.
Arquivos: `internal/db/db.go` (`BackupSQL`, `RestoreSQL`).

## 3) Estrategia de execucao por fase

1. Fase A: manter SQLite em producao e validar branch com codigo multi-dialeto.
2. Fase B: habilitar Postgres em homolog com `HONDAGO_DATABASE_URL`.
3. Fase C: rodar `001`, `002`, `003`, `004` + testes de regressao.
4. Fase D: cutover com rollback documentado.

