# Plano Postgres-Only Final

## Objetivo
Concluir a transição do sistema para operação exclusiva em PostgreSQL, mantendo um snapshot híbrido apenas como contingência, e estabelecer uma base estável para evolução contínua em ambiente web.

## Estratégia Geral
- Manter uma cópia local congelada (`hondago_hibrido`) como fallback operacional.
- Evoluir a pasta principal para `Postgres-only`.
- Publicar em ciclos curtos com validação funcional a cada etapa.

## Etapa 1 | Baseline e segurança operacional
### Escopo
- Criar snapshot local `hondago_hibrido` (somente fallback).
- Congelar snapshot sem novos commits.
- Gerar backup SQL validado do banco atual.

### Critério de aceite
- Snapshot criado e preservado.
- Backup restaurável validado em ambiente de teste.

## Etapa 2 | Configuração única de banco
### Escopo
- Tornar `HONDAGO_DATABASE_URL` obrigatório no app principal.
- Remover fallback implícito para SQLite no fluxo web.
- Atualizar `.env.example`, README e documentação de deploy.

### Critério de aceite
- App não inicia sem `HONDAGO_DATABASE_URL`.
- App inicia corretamente com PostgreSQL.

## Etapa 3 | Inicialização e schema robustos
### Escopo
- Garantir bootstrap no startup: validação de PK, detecção de IDs duplicados e alinhamento de sequence.
- Consolidar SQL de criação/índices Postgres como fonte oficial.

### Critério de aceite
- Instalação nova sobe sem intervenção manual de schema.

## Etapa 4 | Remoção de compatibilidade SQLite no código
### Escopo
- Remover ramificações de compatibilidade que não serão mais usadas.
- Padronizar inserts com `RETURNING id`.
- Eliminar `LastInsertId` no fluxo principal.

### Critério de aceite
- Sem ocorrências ativas no fluxo principal para `sqlite_master`, `julianday`, `LastInsertId`.

## Etapa 5 | Hardening de dados e operação
### Escopo
- Garantir constraints críticas (`PK`, `FK`, `UNIQUE`) nas tabelas principais.
- Criar script pós-restore de sanidade (duplicidade, sequence, constraints).
- Sanitizar mensagens de erro ao usuário final (sem host/IP/porta).

### Critério de aceite
- Smoke técnico aprovado sem erro estrutural de banco.

## Etapa 6 | Go-live controlado e evolução contínua
### Escopo
- Deploy da versão estável no EasyPanel.
- Restore do backup validado.
- Execução de smoke test completo.

### Critério de aceite
- Checklist funcional 100% concluído.
- Operação estável por 2 dias consecutivos.
