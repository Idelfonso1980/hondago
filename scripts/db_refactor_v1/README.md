# DB Refactor v1 (Safe Parallel)

## Objetivo

Criar schema `*_v2` em paralelo, copiar dados e validar, sem alterar tabelas atuais em uso.

## Modos

1. `dry-run`: apenas verifica ambiente e lista tabelas.
2. `apply`: cria `*_v2`, executa backfill e validação.
3. `rollback`: remove tabelas `*_v2`.

## Comandos

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\db_refactor_v1\run_refactor_v1.ps1 -Mode dry-run -DbPath honda.sqlite
powershell -ExecutionPolicy Bypass -File .\scripts\db_refactor_v1\run_refactor_v1.ps1 -Mode apply -DbPath honda.sqlite
powershell -ExecutionPolicy Bypass -File .\scripts\db_refactor_v1\run_refactor_v1.ps1 -Mode rollback -DbPath honda.sqlite
```

Observação:
1. Se `sqlite3` não estiver no PATH, o script usa automaticamente `go run ./scripts/db_refactor_v1/sql_runner.go`.

## Ordem recomendada

1. Finalizar expediente.
2. Fechar app/binário.
3. Rodar `dry-run`.
4. Rodar `apply`.
5. Conferir resultado da validação.
6. Abrir app e validar smoke.
