# Postgres SQL - Ordem de Execução

## Objetivo
Padronizar criação de estrutura e permissões base para o ambiente `Postgres-only`.

## Ordem oficial
1. `001_init.sql`  
Cria tabelas, constraints e índices.

2. `002_seed.sql`  
Cria perfis (`roles`) e permissões (`role_permissions`) iniciais.

3. `003_integridade_ids_pre_backup.sql` (somente em base já populada)  
Saneia duplicidades de `id`, garante PK e realinha sequences antes de gerar backup.

## Execução (psql)
```bash
psql -h <host> -U <user> -d <database> -f docs/sql/postgres/001_init.sql
psql -h <host> -U <user> -d <database> -f docs/sql/postgres/002_seed.sql
psql -h <host> -U <user> -d <database> -f docs/sql/postgres/003_integridade_ids_pre_backup.sql
psql -h <host> -U <user> -d <database> -f docs/sql/postgres/004_pos_restore_sanidade.sql
```

## Execução (Adminer)
1. Importar `001_init.sql`.
2. Importar `002_seed.sql`.
3. (Opcional em base populada) Importar `003_integridade_ids_pre_backup.sql`.
4. Após restore/import, executar `004_pos_restore_sanidade.sql`.

## Validação rápida pós-execução
```sql
SELECT COUNT(*) AS total_tabelas
FROM information_schema.tables
WHERE table_schema = 'public';

SELECT id, name FROM roles ORDER BY id;

SELECT role_id, COUNT(*) AS total_permissoes
FROM role_permissions
GROUP BY role_id
ORDER BY role_id;
```

## Observação
Antes de restaurar dados de backup, manter sempre a ordem acima para garantir integridade de schema.
