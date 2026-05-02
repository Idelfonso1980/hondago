# Go Live Hostinger

Projeto: Honda Go  
Data base: 24/04/2026  
Objetivo: checklist prático para publicar o sistema com segurança, estabilidade e operação controlada.

## 1) Decisão de Publicação

Status recomendado: **Go Live condicionado**.

Publicar somente após:
1. RBAC validado no backend por endpoint.
2. Auditoria de ações sensíveis ativa.
3. Hardening de login/sessão concluído.
4. Backup/restore testado com simulação de desastre.
5. Checklist LGPD mínimo atendido.

## 2) Pré-requisitos de Infra (Hostinger)

1. VPS Linux dedicada (evitar hospedagem compartilhada para app stateful).
2. Usuário de sistema sem privilégio root para rodar o serviço.
3. Reverse proxy (Nginx) com TLS obrigatório.
4. Domínio com certificado SSL válido (Let's Encrypt).
5. Firewall ativo liberando apenas portas necessárias (80/443 e SSH restrito).
6. Timezone padronizada no servidor (preferência UTC, exibição pt-BR na aplicação).

## 3) Segurança da Aplicação

1. Sessão:
- cookie `HttpOnly`, `Secure`, `SameSite=Lax` (ou `Strict` se viável).
- expiração por inatividade.
- logout limpando sessão no servidor.

2. Login:
- lockout progressivo por tentativas inválidas.
- rate limit por IP e por usuário.
- mensagens de erro sem vazar detalhe técnico.

3. Controle de acesso:
- autorização no backend para **todo endpoint sensível**.
- não depender apenas de esconder botão/menu no frontend.

4. Headers de segurança:
- `Content-Security-Policy`
- `X-Frame-Options`
- `X-Content-Type-Options`
- `Referrer-Policy`

5. Entrada/saída:
- validação de payloads no backend.
- sanitização de campos textuais.
- limites de paginação e tamanho de busca.

## 4) Banco de Dados e Dados Críticos

1. Backup diário automático com retenção (ex.: 7 diários, 4 semanais).
2. Restore testado e documentado.
3. Índices revisados para telas de maior uso.
4. Migrações de schema versionadas e reversíveis.
5. CPF e dados sensíveis com controle de acesso por perfil.

## 5) LGPD (mínimo operacional para entrar em produção)

1. Definir base legal de tratamento por funcionalidade.
2. Política de retenção e descarte de dados.
3. Controle de acesso aos dados pessoais (mínimo privilégio).
4. Registro de auditoria para operações sensíveis.
5. Procedimento de incidente de segurança (responsável, prazo, comunicação).
6. Documento de privacidade e termo interno de uso do sistema.

## 6) Operação e Observabilidade

1. Logs estruturados por evento crítico.
2. Métricas mínimas:
- latência média/p95 de rotinas de reserva;
- taxa de sucesso/erro da API;
- falhas de login;
- filas pendentes de mensagens manuais.
3. Healthcheck operacional.
4. Rotina de revisão diária de erro e backlog.

## 7) Pipeline e Qualidade (antes do deploy)

Checklist obrigatório:
1. `go test ./...`
2. `go build ./...`
3. validação UTF-8
4. smoke manual:
- login/logout;
- criar/editar/excluir em CRUD crítico;
- reserva individual e lote;
- dashboard;
- mensagens manuais (copiar e marcar enviada);
- permissões por perfil.

## 8) Rollback

1. Manter binário anterior versionado.
2. Snapshot do banco antes do deploy.
3. Procedimento de rollback com tempo-alvo definido (ex.: 15 min).
4. Checklist pós-rollback (integridade e login).

## 9) Capacidade Inicial (seu cenário)

Cenário informado:
- 150 a 180 usuários solicitando;
- 4 usuários atendendo (execução de reservas para API).

Parecer técnico: **sim, é viável** construir e operar com robustez nesse volume, com as seguintes condições:
1. Limitar concorrência de escrita em rotinas sensíveis.
2. Otimizar queries e paginação nas telas.
3. Monitorar latência da API externa (ponto mais instável da operação).
4. Garantir backup/restore e auditoria antes do go-live.
5. Evoluir para PostgreSQL quando houver crescimento de concorrência de escrita, múltiplas instâncias ou necessidade de escalabilidade horizontal.

Observação prática:
- Para início, o stack atual pode atender bem com disciplina operacional.
- Para escalar com menor risco no médio prazo, planejar migração para PostgreSQL é recomendado.

## 10) Critério Final de Go/No-Go

Go Live somente se:
1. Segurança: itens 3 e 5 concluídos.
2. Operação: itens 6, 7 e 8 concluídos.
3. Teste de carga básico aprovado para o cenário alvo.
4. Responsáveis de negócio e técnico cientes do plano de contingência.

