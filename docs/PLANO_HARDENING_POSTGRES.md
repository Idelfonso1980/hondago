# Plano de Execução Técnico v3

Projeto: Honda Go  
Data: 01/05/2026  
Objetivo: hardening mínimo obrigatório para produção pública + migração controlada de SQLite para PostgreSQL (Square Cloud)

## 1) Objetivo

Elevar o sistema ao nível de segurança mínima operacional para exposição externa e concluir a migração de banco para PostgreSQL com risco controlado, rollback definido e evidências de validação.

## 2) Escopo

1. Hardening de segurança de aplicação e operação.
2. Proteção de segredos e dados sensíveis.
3. Migração técnica de persistência SQLite para PostgreSQL.
4. Go-live com observabilidade, backup/restore e rollback testados.

Fora de escopo nesta versão:
1. Reescrita completa de arquitetura.
2. Escalabilidade horizontal multi-região.
3. Mudanças profundas de UX não relacionadas à segurança/migração.

## 3) Premissas

1. Ambiente-alvo com PostgreSQL gerenciado (Square Cloud).
2. Uso de variáveis de ambiente para segredos.
3. Janela de mudança aprovada para cutover de banco.
4. Equipe com papéis definidos (Engenharia, Infra/DevOps, QA, Segurança, Operação).

## 4) Papéis e Responsáveis

1. Engenharia Backend: implementação de hardening, refatorações e compatibilidade Postgres.
2. Infra/DevOps: variáveis de ambiente, deploy, firewall, TLS/proxy e monitoração.
3. QA: testes funcionais, regressão e carga.
4. Segurança/Compliance: revisão de controles, auditoria e critérios de aceite.
5. Operação: runbook, backup/restore, execução de go-live e rollback.

## 5) Cronograma por Fase

## Fase 0 (D0-D2) - Preparação e Controle de Risco

Objetivo: estabelecer baseline, governança e janela segura.

Itens:
1. Congelar features não críticas durante o ciclo.
Responsável: Engenharia + Produto
Prazo: D0
Evidência: registro de freeze e branch de trabalho (`hardening-postgres`).

2. Definir janela de mudança, critérios de rollback e responsáveis de plantão.
Responsável: Operação + Infra/DevOps
Prazo: D0-D1
Evidência: runbook preliminar com contatos e janela aprovada.

3. Levantar inventário de endpoints sensíveis e dados críticos.
Responsável: Engenharia + Segurança
Prazo: D1-D2
Evidência: matriz endpoint x permissão x risco.

Critério de aceite da fase:
1. Baseline funcional documentado (`login`, `config`, `auth user`, `reserva`, `dashboard`, `backup`).

## Fase 1 (D2-D6) - Hardening Mínimo Obrigatório

Objetivo: eliminar vulnerabilidades básicas para exposição externa.

Itens:
1. Segredos fora de arquivos versionados (`token_principal` via ENV).
Responsável: Engenharia + DevOps
Prazo: D2
Evidência: `config.ini` sem segredo e ENV no ambiente.

2. Remover criação automática de credencial padrão (`admin/admin123`).
Responsável: Engenharia
Prazo: D2-D3
Evidência: fluxo de bootstrap seguro validado.

3. Fortalecer sessão: `Secure`, `HttpOnly`, `SameSite`, expiração absoluta e idle timeout.
Responsável: Engenharia
Prazo: D3
Evidência: teste de cookie/session em browser + teste backend.

4. Implementar proteção CSRF para rotas autenticadas por cookie.
Responsável: Engenharia
Prazo: D3-D4
Evidência: requests sem token CSRF retornando erro esperado.

5. Restringir métodos HTTP por endpoint e retornar `405` para inválidos.
Responsável: Engenharia
Prazo: D4
Evidência: suíte de smoke para método inválido por rota crítica.

6. Uniformizar autorização backend em endpoints sensíveis (`config`, `auth`, `db`, `logs`, `rbac`, `run`).
Responsável: Engenharia + Segurança
Prazo: D4-D5
Evidência: matriz de permissões validada por testes de acesso negativo.

7. Aplicar rate limit em login/MFA e endpoints críticos.
Responsável: Engenharia
Prazo: D5-D6
Evidência: teste de brute force com bloqueio efetivo.

8. Aplicar headers de segurança (`CSP`, `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`).
Responsável: Engenharia
Prazo: D5-D6
Evidência: inspeção de headers via navegador/curl.

Critério de aceite da fase:
1. Nenhum endpoint sensível acessível sem autenticação/permissão.
2. Pentest interno básico sem bypass de sessão/CSRF.

## Fase 2 (D6-D9) - Proteção de Dados Sensíveis

Objetivo: impedir exposição de credenciais em repouso e em logs.

Itens:
1. Criptografar `api_accounts.account_password` com AES-GCM.
Responsável: Engenharia
Prazo: D6-D7
Evidência: valor persistido em ciphertext no banco.

2. Chave de criptografia via variável de ambiente (`HONDAGO_SECRETS_KEY`).
Responsável: DevOps + Engenharia
Prazo: D6-D7
Evidência: chave ausente do código, INI e banco.

3. Mascarar segredos em logs e respostas API.
Responsável: Engenharia
Prazo: D7-D8
Evidência: revisão de logs com redaction aplicada.

4. Fluxo administrativo sem exibição de senha atual (apenas troca/reset).
Responsável: Engenharia + Segurança
Prazo: D8-D9
Evidência: tela e endpoints sem retorno de senha em claro.

Critério de aceite da fase:
1. Dump/backup não expõe credencial em claro.
2. Logs de produção sem segredos.

## Fase 3 (D9-D12) - Preparação Técnica para PostgreSQL

Objetivo: habilitar aplicação para rodar com SQLite e PostgreSQL de forma controlada.

Itens:
1. Introduzir configuração `DATABASE_URL` e driver Postgres (`pgx`).
Responsável: Engenharia
Prazo: D9-D10
Evidência: app iniciando com provider configurável.

2. Ajustar camada de acesso a dados para diferenças de dialeto (placeholders/tipos).
Responsável: Engenharia
Prazo: D10-D11
Evidência: testes de integração em ambos os bancos.

3. Criar migrações versionadas por dialeto (`sqlite`/`postgres`).
Responsável: Engenharia
Prazo: D11-D12
Evidência: scripts idempotentes com rollback.

4. Definir tuning inicial de pool para Postgres.
Responsável: Engenharia + DevOps
Prazo: D11-D12
Evidência: parâmetros documentados e aplicados em homolog.

Critério de aceite da fase:
1. Fluxos críticos funcionando em SQLite e Postgres.

## Fase 4 (D12-D15) - Migração de Schema e Dados

Objetivo: mover dados com consistência e rastreabilidade.

Itens:
1. Provisionar PostgreSQL gerenciado na Square Cloud.
Responsável: DevOps
Prazo: D12
Evidência: instância disponível + credenciais seguras.

2. Aplicar schema e índices no Postgres.
Responsável: Engenharia
Prazo: D12-D13
Evidência: migração executada com sucesso em homolog.

3. Executar carga inicial de dados (SQLite -> Postgres).
Responsável: Engenharia + Operação
Prazo: D13-D14
Evidência: relatório de migração com contagem por tabela.

4. Validar consistência e amostragem de registros sensíveis.
Responsável: QA + Segurança
Prazo: D14-D15
Evidência: checklist de integridade assinado.

Critério de aceite da fase:
1. Paridade funcional homologada e sem perda de dados.

## Fase 5 (D15-D17) - Carga, Operação e Recuperação

Objetivo: validar estabilidade operacional antes do go-live.

Itens:
1. Teste de carga no cenário alvo (180 solicitantes, 3-4 operadores).
Responsável: QA + Engenharia
Prazo: D15-D16
Evidência: relatório com p95, throughput e taxa de erro.

2. Configurar observabilidade mínima (logs estruturados, alertas, healthchecks).
Responsável: DevOps + Engenharia
Prazo: D15-D16
Evidência: dashboards e alertas ativos.

3. Validar backup/restore real no Postgres.
Responsável: Operação + DevOps
Prazo: D16-D17
Evidência: execução de restore com RTO medido.

Critério de aceite da fase:
1. Metas de estabilidade e recuperação atendidas.

## Fase 6 (Go-Live) - Cutover e Rollback

Objetivo: virar produção com risco controlado.

Itens:
1. Freeze curto de escrita e snapshot final do SQLite.
Responsável: Operação
Prazo: Janela de corte
Evidência: snapshot datado.

2. Trocar conexão de produção para Postgres (`DATABASE_URL`).
Responsável: DevOps
Prazo: Janela de corte
Evidência: deploy e status saudável.

3. Executar smoke pós-corte.
Responsável: QA + Engenharia
Prazo: imediata pós-cutover
Evidência: checklist de smoke aprovado.

4. Monitoramento reforçado por 24-48h.
Responsável: Operação + DevOps
Prazo: D+1 a D+2
Evidência: relatório de estabilidade.

5. Rollback (se necessário): voltar conexão para SQLite + restaurar snapshot.
Responsável: Operação + DevOps
Prazo: sob gatilho
Evidência: procedimento executável com RTO <= 15 min.

## 6) Metas de Qualidade e SLO Inicial

1. p95 leitura < 400ms.
2. p95 escrita < 800ms.
3. erro HTTP 5xx < 1%.
4. sucesso de backup diário 100%.
5. restore validado semanalmente.

## 7) Critério Go/No-Go Final

Go-live somente se todos os itens abaixo estiverem verdes:
1. Fases 1 e 2 concluídas (obrigatórias).
2. Migração Postgres validada em homolog (Fases 3 e 4).
3. Teste de carga aprovado (Fase 5).
4. Backup/restore comprovados com evidência.
5. Runbook de operação e rollback assinado.

## 8) Riscos e Mitigações

1. Risco: regressão funcional na troca de banco.
Mitigação: testes de integração multi-banco + smoke completo pós-cutover.

2. Risco: vazamento de segredo por log/config.
Mitigação: ENV-only para segredos + redaction + revisão de logs.

3. Risco: indisponibilidade durante janela de migração.
Mitigação: freeze curto, snapshot pré-corte e rollback validado.

4. Risco: erro de permissão em endpoint crítico.
Mitigação: matriz de autorização + testes de acesso negativo.

## 9) Entregáveis

1. Código com hardening aplicado.
2. Suporte a execução com PostgreSQL.
3. Scripts/migrações versionadas.
4. Runbook de deploy, cutover e rollback.
5. Relatórios de segurança, carga e restore com evidências.

## 10) Quadro Semanal e Plano Diário (D1-D25)

Semana 1 (D1-D5) - Hardening base
1. D1 | Dono: Engenharia + Produto | Entregável: freeze de escopo e baseline funcional | Evidência: checklist baseline assinado.
2. D2 | Dono: Engenharia + DevOps | Entregável: segredos fora de INI/Git + env aplicado | Evidência: `config.ini` sem segredo + variáveis carregadas.
3. D3 | Dono: Engenharia | Entregável: remoção de credencial padrão + bootstrap seguro | Evidência: criação inicial de admin sem senha default.
4. D4 | Dono: Engenharia | Entregável: sessão/cookie endurecidos + timeout | Evidência: testes de expiração e flags de cookie.
5. D5 | Dono: Engenharia + QA | Entregável: CSRF ativo e método HTTP por endpoint | Evidência: testes negativos (`401/403/405`) aprovados.

Marco S1:
1. Nenhum endpoint crítico sem autenticação/permissão.

Semana 2 (D6-D10) - Hardening avançado e proteção de dados
1. D6 | Dono: Engenharia + Segurança | Entregável: revisão RBAC backend rotas críticas | Evidência: matriz permissão x endpoint validada.
2. D7 | Dono: Engenharia | Entregável: rate limit login/MFA/rotas sensíveis | Evidência: teste de brute force com bloqueio.
3. D8 | Dono: Engenharia | Entregável: security headers globais | Evidência: inspeção de headers por curl/navegador.
4. D9 | Dono: Engenharia | Entregável: AES-GCM em `api_accounts.account_password` | Evidência: senha persistida em ciphertext.
5. D10 | Dono: Engenharia + QA | Entregável: redaction de logs + UI sem revelar senha | Evidência: logs sem segredo e tela apenas com reset/troca.

Marco S2:
1. Hardening mínimo concluído.
2. Segredos protegidos em repouso e em trânsito interno.

### Status de Fechamento - Semana 2 (01/05/2026)

1. `Concluído` Revisão RBAC backend em rotas críticas.
Evidência: política de permissão crítica aplicada no backend (`criticalPermissionPolicy`) e validação de acesso por sessão/perfil.

2. `Concluído` Rate limit em login/MFA/endpoints sensíveis.
Evidência: política `endpointRateLimitPolicy` ativa para `/api/app/login`, `/api/app/mfa-login`, `/api/mfa/verify` e rotas críticas de administração.

3. `Concluído` Security headers globais.
Evidência: middleware com headers de segurança (`CSP`, `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`) aplicado nas respostas HTTP.

4. `Concluído` Criptografia AES-GCM para `api_accounts.account_password`.
Evidência: valores persistidos em formato `enc:v1:...` no banco e rotina de descriptografia no uso de autenticação de API.

5. `Concluído` Redaction de segredos em logs/respostas e UI.
Evidência: colunas de senha/token mascaradas na interface de configuração e remoção de debug sensível no retorno da ação "Autenticar".

6. `Concluído` Testes da semana.
Evidência:
- login/logout e sessão com expiração;
- bloqueio por método inválido (`405`) e proteção de acesso;
- CSRF negativo para rotas protegidas por cookie;
- autenticação de usuário API com senha em repouso criptografada e envio em claro somente em memória.

Aceite S2 em 01/05/2026: `APROVADO`.

Semana 3 (D11-D15) - Preparação PostgreSQL
1. D11 | Dono: Engenharia | Entregável: `DATABASE_URL` + provider de banco | Evidência: app sobe por configuração em SQLite/Postgres.
2. D12 | Dono: Engenharia | Entregável: ajustes de SQL/dialeto (placeholders/tipos) | Evidência: testes de integração verdes em ambos bancos.
3. D13 | Dono: Engenharia | Entregável: migrações versionadas `sqlite/postgres` | Evidência: migrações idempotentes com rollback.
4. D14 | Dono: Engenharia + DevOps | Entregável: pool de conexões e timeouts para Postgres | Evidência: parâmetros aplicados em homolog.
5. D15 | Dono: QA + Engenharia | Entregável: smoke multi-banco | Evidência: fluxo crítico aprovado nos dois backends.

Marco S3:
1. Compatibilidade funcional SQLite/Postgres sem regressão crítica.

Semana 4 (D16-D20) - Migração e validação de dados
1. D16 | Dono: DevOps | Entregável: Postgres homolog provisionado (Square Cloud) | Evidência: instância acessível com credenciais seguras.
2. D17 | Dono: Engenharia | Entregável: schema e índices aplicados no Postgres | Evidência: execução de migração sem erro.
3. D18 | Dono: Engenharia + Operação | Entregável: carga inicial SQLite -> Postgres | Evidência: relatório de contagem por tabela.
4. D19 | Dono: QA + Segurança | Entregável: validação de consistência e amostragem | Evidência: checklist de integridade assinado.
5. D20 | Dono: QA + Engenharia | Entregável: regressão funcional completa em homolog | Evidência: testes de CRUD/reserva/dashboard/MFA aprovados.

Marco S4:
1. Dados íntegros no Postgres e paridade funcional homologada.

Semana 5 (D21-D25) - Carga, operação e go-live
1. D21 | Dono: QA + Engenharia | Entregável: teste de carga cenário alvo | Evidência: relatório com p95 e taxa de erro.
2. D22 | Dono: DevOps + Engenharia | Entregável: observabilidade mínima ativa | Evidência: logs estruturados + alertas + healthchecks.
3. D23 | Dono: Operação + DevOps | Entregável: backup/restore real validado | Evidência: restore executado com RTO registrado.
4. D24 | Dono: Operação + DevOps + Engenharia | Entregável: ensaio geral de cutover e rollback | Evidência: simulação concluída com checklist.
5. D25 | Dono: Operação + QA + Engenharia | Entregável: cutover de produção + smoke + monitoramento 24-48h | Evidência: relatório de estabilidade pós-go-live.

Marco S5:
1. Go-live concluído com rollback validado e operação estável.
