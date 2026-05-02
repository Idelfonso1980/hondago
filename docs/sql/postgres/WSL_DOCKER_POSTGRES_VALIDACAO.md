# Validação local (WSL + Docker)

## 1) Subir Postgres + Adminer

No WSL, na raiz do projeto:

```bash
docker compose up -d
docker compose ps
```

URLs:
- Postgres: `localhost:5432`
- Adminer: `http://localhost:8081`

Credenciais padrão do compose:
- DB: `hondago`
- User: `hondago`
- Password: `hondago_dev_password`

## 2) Criar schema e seed

```bash
docker exec -i hondago-postgres psql -U hondago -d hondago < docs/sql/postgres/001_init.sql
docker exec -i hondago-postgres psql -U hondago -d hondago < docs/sql/postgres/002_seed.sql
```

## 3) Exportar CSV do SQLite (Windows PowerShell)

```powershell
powershell -ExecutionPolicy Bypass -File .\docs\sql\postgres\004_export_sqlite_to_csv.ps1 -DbPath .\honda.db -OutDir .\tmp\sqlite_csv
```

## 4) Copiar CSV para WSL e importar no Postgres

No WSL:

```bash
mkdir -p /tmp/hondago_csv
cp -r /mnt/c/Users/Idelfonso/Downloads/bkp\ Go/Honda\ Go/tmp/sqlite_csv/* /tmp/hondago_csv/
```

Entrar no `psql`:

```bash
docker exec -it hondago-postgres psql -U hondago -d hondago
```

No prompt do `psql`, rode:

```sql
\set csv_dir '/tmp/hondago_csv'
\i /workspace/docs/sql/postgres/003_migrate_from_sqlite.sql
```

Observação:
- Se `/workspace` não existir no container, use opção B abaixo.

Opção B (sem `\i`):
- Copie o SQL para dentro do container e rode:

```bash
docker cp docs/sql/postgres/003_migrate_from_sqlite.sql hondago-postgres:/tmp/003_migrate_from_sqlite.sql
docker exec -it hondago-postgres psql -U hondago -d hondago -c "\set csv_dir '/tmp/hondago_csv'" -f /tmp/003_migrate_from_sqlite.sql
```

## 5) Apontar aplicação para Postgres

No `.env`:

```env
HONDAGO_DATABASE_URL=postgres://hondago:hondago_dev_password@localhost:5432/hondago?sslmode=disable
```

## 6) Rebuild e subir aplicação

No PowerShell (raiz do projeto):

```powershell
go build -a -o .\honda-go-gui.exe .\cmd\honda-gui
.\honda-go-gui.exe
```

## 7) Smoke test mínimo

1. Login no sistema.
2. Configuração de usuários -> autenticar (deve retornar sucesso para usuários válidos).
3. Dashboard carregando dados.
4. Fluxo de solicitação/reserva.
5. Logs sem segredo em claro.

## 8) Limpeza ambiente

```bash
docker compose down
```

Para remover dados também:

```bash
docker compose down -v
```
