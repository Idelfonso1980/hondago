# Smoke Test Pós-Migração para Postgres

## 1. Pré-condições
- [x] Container Postgres em execução (`docker ps` com `hondago-postgres`).
- [x] `.env` apontando para Postgres (`HONDAGO_DATABASE_URL` preenchido).
- [x] Executável atualizado (`honda-go-gui.exe`) da última build.
- [x] Login no sistema com perfil `admin`.

## 2. Sanidade inicial
- [x] Abrir Dashboard sem erro em status.
- [x] Confirmar cards carregando valores.
- [x] Confirmar nome do usuário no topo e sessão autenticada.

## 3. Telas administrativas
- [x] `Configuração > Configurações de Usuário`: listar, editar, salvar e autenticar usuário.
- [x] `Configuração > Usuários do Sistema`: listar, editar, salvar e excluir teste.
- [x] `Configuração > Controle de Acessos (RBAC)`: abrir, alterar 1 permissão de teste e salvar.
- [x] `Configuração > Auditoria`: listar registros sem erro de scan/SQL.

## 4. Cadastros de apoio
- [x] `Configuração > Modelos`: buscar, editar e salvar.
- [x] `Configuração > Produtos`: buscar, editar e salvar.
- [x] `Configuração > Assembleia`: listar, editar e salvar.
- [x] `Configuração > Grupos Ativos`: listar, editar e salvar.
- [x] `Configuração > Ids Grupos Disponíveis`: listar com paginação e coluna `Parcelas` preenchida quando aplicável.

## 5. Operação de reservas
- [x] `Painel de Reservas > Solicitações`: carregar sem erro.
- [x] `Nova Solicitação`: ao preencher grupo, autopreencher `Qtde Parcelas` e `Perc. Lance`.
- [x] `Solicitar Cota`: mesmo comportamento de autopreenchimento.
- [x] Reservar 1 solicitação de teste e validar retorno de sucesso.

## 6. Mensagens e cotas
- [x] `Mensagens`: listar sem erro, copiar mensagem com ícone correto (copiar).
- [x] `Cotas Reservadas`: listar e excluir 1 registro de teste.

## 7. Segurança mínima (regressão)
- [x] Senhas/tokens mascarados na interface de configuração de usuários.
- [x] `api_accounts.account_password` permanece criptografado (`enc:v1:`) no banco.
- [x] Headers e sessão permanecem estáveis (sem logout inesperado durante navegação).

## 8. Consulta rápida no Postgres (opcional)
- [x] Contagens básicas:
  - `roles` > 0
  - `users` > 0
  - `api_accounts` > 0
  - `requests` > 0
- [x] Consulta simples em `active_groups` com `group_code`, `due_day`, `term_months`, `participants_count`, `first_assembly_date`.

## 9. Critério de aceite
- [x] Nenhuma tela crítica com erro SQL.
- [x] Fluxo de autenticação + reserva funcionando ponta a ponta.
- [x] Cadastros principais (Modelos/Produtos/Assembleia/Grupos) editáveis.
- [x] Auditoria e Mensagens funcionando.

## 10. Evidências recomendadas
- [x] Print do Dashboard carregado.
- [x] Print de cada tela administrativa validada.
- [x] Print de reserva concluída.
- [x] Print da tabela `api_accounts` com `account_password` criptografado.

## 11. Bloqueios e pendencias
- [x] Bloqueio externo ativo: API Honda offline ate 08:00 (horario local).
- [x] Item pendente por bloqueio: `5. Operacao de reservas > Reservar 1 solicitacao de teste e validar retorno de sucesso`.
- [x] Item pendente por bloqueio: `9. Criterio de aceite > Fluxo de autenticacao + reserva funcionando ponta a ponta`.

### Roteiro de reteste (08:00)
- [x] Autenticar 1 usuario em `Configuracao > Configuracoes de Usuario`.
- [x] Reservar 1 solicitacao em `Painel de Reservas > Solicitacoes`.
- [x] Validar retorno de sucesso e presenca do registro em `Cotas Reservadas`.
- [x] Marcar os 2 itens pendentes como concluidos.
