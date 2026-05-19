# Plano de Hardening RBAC Backend (Mapeamento + Execução)

Data: 2026-05-12
Escopo: `cmd/honda-gui/main.go` (API HTTP), regras de autorização por permissão e escopo de dados.

## 1. Objetivo

Eliminar pontos onde a proteção depende mais da interface (front-end) do que do backend.

Resultado esperado:
- Toda ação sensível validada no backend por permissão explícita.
- Escopo por perfil (filial/usuário) aplicado no backend de forma consistente.
- Redução de risco de chamada direta de API com sessão válida.

## 2. Estado Atual (Resumo)

Hoje o middleware `withAuth` aplica sessão + CSRF para rotas `/api/*` e só bloqueia por permissão automática para endpoints listados em `criticalPermissionPolicy`.

Arquivo referência:
- `cmd/honda-gui/main.go` (`methodPolicy`, `criticalPermissionPolicy`, `withAuth`, handlers).

## 3. Mapeamento de Risco por Endpoint

### 3.1 Alto risco (mutação sem trava forte central por permissão)

- `/api/solicitacoes/save`
- `/api/solicitacoes/delete`
- `/api/solicitacoes/reservar`
- `/api/solicitacoes/reservar-batch`
- `/api/stop`

Observação: ficam cobertos por sessão/CSRF, porém sem exigência central de permissão no `criticalPermissionPolicy` e sem padrão unificado de `requirePermission` nesses handlers.

### 3.2 Médio risco (config operacional sem política crítica explícita)

- `active_groups`: search/get/parcelas/similares/save/delete/delete-batch
- `available_group_ids`: search/get/check/save/delete/delete-batch
- `models`: search/get/save/delete
- `produtos`: search/get/save/delete
- `assembleias`: search/get/perclance/save/delete
- `cotasreservadas`: delete/delete-batch
- `config/load` (avaliar leitura por perfil)

### 3.3 Baixo risco / já robusto

- Banco (`db:*`)
- Logs e diagnósticos (`logs:*`)
- Auditoria (`audit:view`)
- RBAC (`roles:manage`)
- Política de senha (`config:password_policy`)
- Gestão de usuários app (`users:read/create/edit/delete`)

## 4. Plano de Execução (Fases)

## Fase 1 - Travas de permissão em endpoints mutáveis (rápida, baixo impacto)

1. Adicionar ao `criticalPermissionPolicy` os endpoints mutáveis faltantes, começando por:
- `solicitacoes:save` -> `solicitacoes:create`/`solicitacoes:edit` (ver nota abaixo)
- `solicitacoes:delete` -> `solicitacoes:delete`
- `solicitacoes:reservar` -> `cotas:reserve`
- `solicitacoes:reservar-batch` -> `cotas:reserve`
- `cotasreservadas:delete/delete-batch` -> permissão específica (sugerida: `cotas:reserve` ou nova `cotas:delete_reserved`)
- `/api/stop` -> nova permissão (sugerida: `system:stop`) ou `configs:manage`

Nota: para `save`, como cria e edita no mesmo endpoint, validar no handler: create exige `solicitacoes:create`, update exige `solicitacoes:edit`.

2. Reforçar no handler mutável com `requirePermission(...)` explícito (defesa em profundidade).

## Fase 2 - Escopo por perfil no backend (consistência)

Aplicar política de escopo uniforme para perfis não-admin/super_usuario:
- Restringir leitura/edição conforme filial e/ou ownership quando aplicável.
- Padronizar em `solicitacoes`, `cotasreservadas`, `notificacoes`, etc.

## Fase 3 - Revisão de leitura de configuração

- Avaliar `config/load` e separar dados sensíveis de dados de UI.
- Exigir permissão específica para blocos sensíveis (tokens, parâmetros de integração, etc.).

## Fase 4 - Observabilidade e auditoria

- Garantir log de acesso negado com `username`, `role`, `permission`, `endpoint`.
- Auditar ações críticas (delete, reserve, alterações de cadastro/configuração).

## 5. Estratégia de Teste

### 5.1 Matriz de perfis

Testar ao menos:
- `admin`
- `super_usuario`
- `gerente`
- `supervisor`
- `operador`
- `vendedor`
- `viewer`

### 5.2 Casos mínimos

1. UI permite e API permite (cenário esperado).
2. UI bloqueia e API bloqueia (cenário esperado).
3. Chamada direta de API por perfil sem permissão retorna `403`.
4. Escopo de filial/usuário respeitado em leitura e mutação.

### 5.3 Regressão funcional

- Fluxo de login/mfa/troca de senha.
- Fluxo completo de solicitação e reserva.
- Configurações administrativas para admin.

## 6. Risco e Mitigação

Risco: baixo a médio (quebra por bloquear fluxo legítimo de algum perfil).
Mitigação:
1. Implementar em commit isolado de segurança.
2. Habilitar por fases e validar perfil a perfil.
3. Manter rollback simples (reversão de um commit).

## 7. Entregáveis

1. Patch backend de hardening RBAC (sem mudanças visuais).
2. Tabela de endpoints com permissão exigida.
3. Evidência de testes por perfil (checklist).

## 8. Próximo passo recomendado

Executar Fase 1 imediatamente (ganho alto de segurança com baixo impacto).
