# Plano de Execução Técnico v2

Projeto: Honda Go  
Data: 28/04/2026  
Ambiente-alvo: HostGator (preferencialmente VPS Linux com Nginx + TLS)

## 1) Objetivo

Levar o sistema para produção com segurança, robustez e escalabilidade compatíveis com o cenário informado:
- ~180 usuários solicitando cotas (uso distribuído ao longo do dia).
- 3 a 4 atendentes executando reserva e autenticação de contas API.

## 2) Diagnóstico Técnico Atual (baseado no código)

### Pontos fortes

1. Backend Go com boa performance para I/O e concorrência (`internal/engine/*`).
2. Controle de sessão no backend e persistência em tabela `app_sessions`.
3. RBAC, auditoria e MFA já existem no sistema.
4. SQLite em WAL com `busy_timeout`, além de índices operacionais básicos.
5. Rotinas de backup/restore e trilha de log operacional.

### Lacunas críticas para produção pública

1. Exposição de serviço em `0.0.0.0:8787` por padrão.
2. Cookie de sessão sem `Secure` e sem expiração/idle-timeout efetivo.
3. Ausência de proteção CSRF para endpoints autenticados por cookie.
4. Endpoints administrativos e sensíveis ainda sem política uniforme de autorização por permissão.
5. Credenciais sensíveis de `api_accounts` armazenadas em texto claro no banco.
6. Falta de padronização rígida de método HTTP por endpoint (GET/POST/DELETE).
7. SQLite com `MaxOpenConns=1` limita throughput de escrita concorrente e pode virar gargalo no crescimento.

## 3) Parecer de Go-Live para HostGator

Status recomendado: **Go-Live Condicionado**.

Pode operar no volume atual, **desde que** os itens críticos de segurança sejam concluídos antes da abertura externa.

### Recomendação de infraestrutura

1. HostGator VPS (não compartilhado) com Ubuntu LTS.
2. App Go atrás de Nginx (reverse proxy) em `127.0.0.1:8787`.
3. HTTPS obrigatório com Let's Encrypt.
4. Firewall liberando somente `80/443` e SSH restrito por IP/chave.
5. Execução via `systemd` com usuário de serviço sem privilégios.

## 4) Meta de Capacidade (cenário atual)

1. Suportar 180 usuários de front com picos curtos sem indisponibilidade.
2. Suportar 3-4 atendentes com operações simultâneas de reserva/API.
3. Meta de API interna:
- p95 leitura < 400ms (consultas comuns)
- p95 escrita < 800ms (salvar/editar)
- erro HTTP 5xx < 1%

Observação: no curto prazo SQLite atende; para escala maior e múltiplas instâncias, planejar migração para PostgreSQL.

## 5) Plano de Execução por Fase

## Fase 0 - Hardening obrigatório (D0-D3)

1. Forçar execução atrás de proxy TLS.
2. Cookie de sessão com `Secure`, `HttpOnly`, `SameSite=Lax` e `MaxAge`.
3. Expiração de sessão por inatividade + revogação no logout.
4. Implementar proteção CSRF para todas as rotas `/api/*` autenticadas por cookie.
5. Definir allowlist de métodos HTTP por endpoint.
6. Revisar autorização em todos endpoints sensíveis (config, auth users, run/auth, db ops, logs, auditoria).

Critério de aceite:
1. Pentest interno básico sem bypass de sessão/CSRF.
2. Nenhum endpoint sensível acessível sem permissão adequada.

## Fase 1 - Segurança de dados e credenciais (D4-D7)

1. Criptografar `api_accounts.account_password` em repouso (AES-GCM com chave em variável de ambiente).
2. Mascarar dados sensíveis em logs e respostas de API.
3. Ajustar política de senha (complexidade mínima e rotação para contas administrativas).
4. Implementar rate limit por IP e por usuário em login e endpoints críticos.

Critério de aceite:
1. Nenhuma credencial em claro em dumps/logs.
2. Bloqueio efetivo de brute force.

## Fase 2 - Robustez operacional (Semana 2)

1. Healthcheck de aplicação (`/healthz`) e readiness.
2. Logs estruturados (JSON lines) com `request_id`, `user_id`, `endpoint`, `latency_ms`, `status`.
3. Rotação/retenção de logs.
4. Backup automático diário + restore validado (teste de restauração real).
5. Procedimento de rollback com RTO de 15 minutos.

Critério de aceite:
1. Restore testado em ambiente clone.
2. Diagnóstico de falha executável sem acessar código.

## Fase 3 - Escalabilidade e banco (Semana 3-4)

1. Revisar queries mais pesadas (dashboard e listagens com filtro).
2. Adicionar índices faltantes por consulta real (via profiling).
3. Implementar paginação consistente em endpoints de listagem.
4. Definir plano de migração PostgreSQL (sem executar ainda), incluindo:
- modelo de dados compatível
- estratégia de cutover
- plano de rollback

Critério de aceite:
1. p95 dentro da meta no teste de carga do cenário de 180 usuários.
2. Sem lock prolongado de banco em operações comuns.

## Fase 4 - Governança de release (Semana 5)

1. Pipeline obrigatório: `go test`, `go build`, smoke e checks de segurança.
2. Checklist de deploy/rollback e operação diária.
3. Matriz de permissões auditável por perfil.
4. Revisão de LGPD mínima (acesso, retenção, descarte e incidente).

Critério de aceite:
1. Processo de release reproduzível.
2. Abertura de produção com checklist assinado por técnico e operação.

## 6) Ordem de Implementação Recomendada no Código

1. `cmd/honda-gui/main.go`
- middleware de segurança (CSRF, headers, método HTTP, sessão com expiração)
- revisão uniforme de `requirePermission`/`requireAdmin` por endpoint

2. `internal/db/db.go`
- criptografia de credenciais sensíveis
- ajustes de sessão (expiração e invalidação)

3. `internal/db/schema.go`
- campos de apoio para expiração de sessão, auditoria e governança
- índices adicionais dirigidos por consulta

4. `docs/*`
- runbook operacional
- checklist de deploy/rollback

## 7) Riscos e Mitigações

1. Risco: acesso indevido por falha de autorização em endpoint isolado.  
Mitigação: matriz de permissões obrigatória + teste automatizado de autorização.

2. Risco: sequestro de sessão em tráfego sem TLS.  
Mitigação: HTTPS obrigatório + cookie `Secure` + timeout de sessão.

3. Risco: indisponibilidade por contenção SQLite em escrita.  
Mitigação: otimizar transações/índices, reduzir operações longas e preparar migração para PostgreSQL.

4. Risco: perda de dados por restore não validado.  
Mitigação: backup diário + restore testado semanalmente.

## 8) Go/No-Go Final

Go-Live na HostGator somente se todos os itens abaixo estiverem verdes:

1. Fase 0 concluída (obrigatória).
2. Fase 1 concluída (obrigatória).
3. Teste de carga do cenário real aprovado.
4. Backup e restore validados com evidência.
5. Checklist de operação e rollback assinado.

## 9) Próximos passos imediatos (48h)

1. Endurecer sessão/cookie/CSRF e revisar permissões dos endpoints mais críticos.
2. Publicar ambiente VPS com Nginx + TLS + firewall.
3. Executar teste de carga de baseline antes de abrir acesso externo.
